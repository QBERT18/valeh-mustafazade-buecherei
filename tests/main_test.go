package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/QBERT18/valeh-mustafazade-buecherei/api"
	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/QBERT18/valeh-mustafazade-buecherei/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

// The tests run against the real Postgres and RustFS from docker-compose, but
// in their own database and bucket so the dev data is never touched.
const (
	testDBName = "buecherei_test"
	testBucket = "books-test"
)

var (
	testDB      *gorm.DB
	testStorage *storage.Storage
	router      *gin.Engine
)

func TestMain(m *testing.M) {
	_ = godotenv.Load("../.env")
	gin.SetMode(gin.TestMode)

	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "test setup failed: %v\nIs the stack running? Start it with `docker compose up -d`.\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func setup() error {
	// Connect to the dev database once, only to create the test database.
	admin, err := db.Open()
	if err != nil {
		return err
	}
	var exists bool
	if err := admin.Raw("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = ?)", testDBName).Scan(&exists).Error; err != nil {
		return err
	}
	if !exists {
		if err := admin.Exec("CREATE DATABASE " + testDBName).Error; err != nil {
			return err
		}
	}
	if sqlDB, err := admin.DB(); err == nil {
		sqlDB.Close()
	}

	os.Setenv("POSTGRES_DB", testDBName)
	if testDB, err = db.Open(); err != nil {
		return err
	}
	if err := db.Migrate(testDB); err != nil {
		return err
	}

	st, err := storage.Open() // checks the dev bucket; we reuse its client
	if err != nil {
		return err
	}
	ctx := context.Background()
	ok, err := st.Client.BucketExists(ctx, testBucket)
	if err != nil {
		return err
	}
	if !ok {
		if err := st.Client.MakeBucket(ctx, testBucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	testStorage = &storage.Storage{Client: st.Client, Bucket: testBucket}

	router = gin.New()
	api.Register(router, testDB, testStorage)
	return nil
}

// reset empties all tables and the test bucket.
func reset(t *testing.T) {
	t.Helper()
	if err := testDB.Exec("TRUNCATE loans, people, books, shelves RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for obj := range testStorage.Client.ListObjects(ctx, testBucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			t.Fatal(obj.Err)
		}
		if err := testStorage.Client.RemoveObject(ctx, testBucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			t.Fatal(err)
		}
	}
}

// --- HTTP helpers ---

func send(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// call sends a request with an optional JSON body and checks the status code.
func call(t *testing.T, method, path string, body any, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := send(req)
	if w.Code != wantStatus {
		t.Fatalf("%s %s: status %d, want %d, body: %s", method, path, w.Code, wantStatus, w.Body)
	}
	return w
}

func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decoding %s: %v", w.Body, err)
	}
	return v
}

func errorMessage(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return decode[map[string]string](t, w)["error"]
}

// --- fixtures ---

func createShelf(t *testing.T, name string, limit int) db.Shelf {
	t.Helper()
	w := call(t, "POST", "/shelves", gin.H{"ShelfName": name, "Limit": limit}, http.StatusCreated)
	return decode[db.Shelf](t, w)
}

func createBook(t *testing.T, title string, shelfID *uint) db.Book {
	t.Helper()
	w := call(t, "POST", "/books", gin.H{"Titel": title, "ShelfID": shelfID}, http.StatusCreated)
	return decode[db.Book](t, w)
}

func createPerson(t *testing.T, name string) db.Person {
	t.Helper()
	w := call(t, "POST", "/people", gin.H{"Name": name}, http.StatusCreated)
	return decode[db.Person](t, w)
}

func lend(t *testing.T, bookID, personID uint) db.Loan {
	t.Helper()
	w := call(t, "POST", fmt.Sprintf("/books/%d/loan", bookID), gin.H{"PersonID": personID}, http.StatusCreated)
	return decode[db.Loan](t, w)
}
