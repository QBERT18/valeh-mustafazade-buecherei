package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/minio/minio-go/v7"
)

// testPDF is a real one-page PDF with one line of text.
var testPDF = mustReadFile("testdata/test.pdf")

func mustReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

// testEPUB builds a minimal EPUB: a zip whose first, uncompressed entry is "mimetype".
func testEPUB(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("application/epub+zip"))
	w, _ = zw.Create("content.xml")
	w.Write([]byte("<x/>"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func upload(t *testing.T, bookID uint, filename string, content []byte, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if filename != "" {
		fw, err := mw.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(content)
	}
	mw.Close()

	req := httptest.NewRequest("PUT", fmt.Sprintf("/books/%d/file", bookID), &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := send(req)
	if w.Code != wantStatus {
		t.Fatalf("upload %q: status %d, want %d, body: %s", filename, w.Code, wantStatus, w.Body)
	}
	return w
}

func bucketKeys(t *testing.T) []string {
	t.Helper()
	var keys []string
	for obj := range testStorage.Client.ListObjects(context.Background(), testBucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			t.Fatal(obj.Err)
		}
		keys = append(keys, obj.Key)
	}
	return keys
}

func TestUploadAndDownloadBookFile(t *testing.T) {
	reset(t)
	book := createBook(t, "Faust", nil)

	got := decode[db.Book](t, upload(t, book.ID, "faust.pdf", testPDF, http.StatusOK))
	if got.FileName != "faust.pdf" || got.FileContentType != "application/pdf" || got.FileSize != int64(len(testPDF)) {
		t.Errorf("unexpected file fields: %+v", got)
	}
	keys := bucketKeys(t)
	if len(keys) != 1 || !strings.HasPrefix(keys[0], fmt.Sprintf("books/%d/", book.ID)) || !strings.HasSuffix(keys[0], ".pdf") {
		t.Fatalf("bucket keys = %v, want one books/%d/<uuid>.pdf", keys, book.ID)
	}

	w := call(t, "GET", fmt.Sprintf("/books/%d/file", book.ID), nil, http.StatusOK)
	if !bytes.Equal(w.Body.Bytes(), testPDF) {
		t.Error("downloaded file differs from the upload")
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); cd != "attachment; filename=faust.pdf" {
		t.Errorf("Content-Disposition = %q", cd)
	}

	t.Run("replace removes the old file", func(t *testing.T) {
		got := decode[db.Book](t, upload(t, book.ID, "Die Verwandlung – Kafka.epub", testEPUB(t), http.StatusOK))
		if got.FileContentType != "application/epub+zip" {
			t.Errorf("FileContentType = %q", got.FileContentType)
		}
		newKeys := bucketKeys(t)
		if len(newKeys) != 1 || newKeys[0] == keys[0] || !strings.HasSuffix(newKeys[0], ".epub") {
			t.Errorf("bucket keys = %v, want only the new .epub", newKeys)
		}
		w := call(t, "GET", fmt.Sprintf("/books/%d/file", book.ID), nil, http.StatusOK)
		if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "filename*=utf-8''Die%20Verwandlung%20%E2%80%93%20Kafka.epub") {
			t.Errorf("Content-Disposition = %q", cd)
		}
	})
}

func TestUploadBookFileRejects(t *testing.T) {
	reset(t)
	book := createBook(t, "Faust", nil)

	upload(t, book.ID, "notes.txt", []byte("hello"), http.StatusUnsupportedMediaType)
	upload(t, book.ID, "", nil, http.StatusBadRequest)
	upload(t, 999, "faust.pdf", testPDF, http.StatusNotFound)

	big := make([]byte, 50<<20+1) // 1 byte over the 50 MB limit
	copy(big, "%PDF-1.4")
	upload(t, book.ID, "big.pdf", big, http.StatusRequestEntityTooLarge)

	if keys := bucketKeys(t); len(keys) != 0 {
		t.Errorf("rejected uploads left files in the bucket: %v", keys)
	}
}

func TestDeleteBookFile(t *testing.T) {
	reset(t)
	book := createBook(t, "Faust", nil)
	upload(t, book.ID, "faust.pdf", testPDF, http.StatusOK)
	path := fmt.Sprintf("/books/%d/file", book.ID)

	call(t, "DELETE", path, nil, http.StatusNoContent)
	if keys := bucketKeys(t); len(keys) != 0 {
		t.Errorf("file still in bucket: %v", keys)
	}
	got := decode[db.Book](t, call(t, "GET", fmt.Sprintf("/books/%d", book.ID), nil, http.StatusOK))
	if got.FileName != "" || got.FileSize != 0 {
		t.Errorf("file fields not cleared: %+v", got)
	}

	call(t, "GET", path, nil, http.StatusNotFound)
	call(t, "DELETE", path, nil, http.StatusNotFound)
	call(t, "DELETE", "/books/999/file", nil, http.StatusNotFound)
}
