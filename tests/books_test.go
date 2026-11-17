package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
)

func TestCreateBook(t *testing.T) {
	reset(t)
	shelf := createShelf(t, "A", 1)

	w := call(t, "POST", "/books", gin.H{
		"Titel": "Faust", "Autor": "Goethe", "ISBN": "123", "Jahr": "1808-01-01T00:00:00Z", "ShelfID": shelf.ID,
	}, http.StatusCreated)
	book := decode[db.Book](t, w)
	if book.ID == 0 || book.Titel != "Faust" || book.Autor != "Goethe" || book.ISBN != "123" {
		t.Errorf("unexpected book: %+v", book)
	}
	if time.Time(book.Jahr).Year() != 1808 {
		t.Errorf("Jahr = %v, want 1808", time.Time(book.Jahr))
	}
	if book.ShelfID == nil || *book.ShelfID != shelf.ID {
		t.Errorf("ShelfID = %v, want %d", book.ShelfID, shelf.ID)
	}

	t.Run("without shelf", func(t *testing.T) {
		if b := createBook(t, "Ohne Regal", nil); b.ShelfID != nil {
			t.Errorf("ShelfID = %d, want nil", *b.ShelfID)
		}
	})
	t.Run("Präsenzbestand", func(t *testing.T) {
		w := call(t, "POST", "/books", gin.H{"Titel": "Brockhaus", "Praesenzbestand": true}, http.StatusCreated)
		if !decode[db.Book](t, w).Praesenzbestand {
			t.Error("Praesenzbestand was not saved")
		}
	})
	t.Run("missing title", func(t *testing.T) {
		call(t, "POST", "/books", gin.H{"Autor": "Goethe"}, http.StatusBadRequest)
	})
	t.Run("unknown shelf", func(t *testing.T) {
		w := call(t, "POST", "/books", gin.H{"Titel": "X", "ShelfID": 999}, http.StatusBadRequest)
		if msg := errorMessage(t, w); msg != "shelf not found" {
			t.Errorf("error = %q", msg)
		}
	})
	t.Run("full shelf", func(t *testing.T) {
		w := call(t, "POST", "/books", gin.H{"Titel": "X", "ShelfID": shelf.ID}, http.StatusConflict)
		if msg := errorMessage(t, w); msg != "shelf is full" {
			t.Errorf("error = %q", msg)
		}
	})
}

func TestListBooks(t *testing.T) {
	reset(t)
	createBook(t, "Faust", nil)
	createBook(t, "Effi Briest", nil)

	w := call(t, "GET", "/books", nil, http.StatusOK)
	if books := decode[[]db.Book](t, w); len(books) != 2 {
		t.Fatalf("got %d books, want 2", len(books))
	}
	if strings.Contains(w.Body.String(), "FileKey") {
		t.Error("FileKey must not be exposed in the API")
	}
}

func TestGetBook(t *testing.T) {
	reset(t)
	shelf := createShelf(t, "A", 3)
	book := createBook(t, "Faust", &shelf.ID)

	got := decode[db.Book](t, call(t, "GET", fmt.Sprintf("/books/%d", book.ID), nil, http.StatusOK))
	if got.Titel != "Faust" || got.Shelf == nil || got.Shelf.ShelfName != "A" {
		t.Errorf("unexpected book (shelf should be included): %+v", got)
	}

	call(t, "GET", "/books/999", nil, http.StatusNotFound)
	call(t, "GET", "/books/abc", nil, http.StatusBadRequest)
}

func TestUpdateBook(t *testing.T) {
	reset(t)
	full := createShelf(t, "Voll", 1)
	free := createShelf(t, "Frei", 5)
	occupant := createBook(t, "Belegt", &full.ID)
	book := createBook(t, "Faust", &free.ID)
	path := fmt.Sprintf("/books/%d", book.ID)

	got := decode[db.Book](t, call(t, "PUT", path, gin.H{
		"Titel": "Faust I", "Autor": "Goethe", "ShelfID": free.ID,
	}, http.StatusOK))
	if got.Titel != "Faust I" || got.Autor != "Goethe" {
		t.Errorf("update not applied: %+v", got)
	}

	t.Run("move to full shelf", func(t *testing.T) {
		call(t, "PUT", path, gin.H{"Titel": "Faust I", "ShelfID": full.ID}, http.StatusConflict)
	})
	t.Run("staying on a full shelf is fine", func(t *testing.T) {
		call(t, "PUT", fmt.Sprintf("/books/%d", occupant.ID), gin.H{"Titel": "Belegt (neu)", "ShelfID": full.ID}, http.StatusOK)
	})
	t.Run("take off shelf", func(t *testing.T) {
		if got := decode[db.Book](t, call(t, "PUT", path, gin.H{"Titel": "Faust I"}, http.StatusOK)); got.ShelfID != nil {
			t.Errorf("ShelfID = %d, want nil", *got.ShelfID)
		}
	})
	t.Run("cannot become Präsenzbestand while on loan", func(t *testing.T) {
		lend(t, book.ID, createPerson(t, "Anna").ID)
		w := call(t, "PUT", path, gin.H{"Titel": "Faust I", "Praesenzbestand": true}, http.StatusConflict)
		if msg := errorMessage(t, w); msg != "book is on loan" {
			t.Errorf("error = %q", msg)
		}
	})
	t.Run("validation and not found", func(t *testing.T) {
		call(t, "PUT", path, gin.H{"Autor": "no title"}, http.StatusBadRequest)
		call(t, "PUT", "/books/999", gin.H{"Titel": "X"}, http.StatusNotFound)
	})
}

func TestDeleteBook(t *testing.T) {
	reset(t)
	book := createBook(t, "Faust", nil)
	path := fmt.Sprintf("/books/%d", book.ID)

	call(t, "DELETE", path, nil, http.StatusNoContent)
	call(t, "GET", path, nil, http.StatusNotFound)
	call(t, "DELETE", path, nil, http.StatusNotFound)

	t.Run("on loan", func(t *testing.T) {
		lent := createBook(t, "Effi Briest", nil)
		lend(t, lent.ID, createPerson(t, "Anna").ID)
		call(t, "DELETE", fmt.Sprintf("/books/%d", lent.ID), nil, http.StatusConflict)
	})
	t.Run("deleted book frees its shelf spot", func(t *testing.T) {
		shelf := createShelf(t, "A", 1)
		b := createBook(t, "X", &shelf.ID)
		call(t, "DELETE", fmt.Sprintf("/books/%d", b.ID), nil, http.StatusNoContent)
		createBook(t, "Y", &shelf.ID)
	})
}

// Two requests must not both take the last spot on a shelf.
func TestShelfLimitUnderConcurrency(t *testing.T) {
	reset(t)
	shelf := createShelf(t, "A", 1)

	const n = 10
	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(gin.H{"Titel": fmt.Sprintf("Buch %d", i), "ShelfID": shelf.ID})
			req := httptest.NewRequest("POST", "/books", strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			codes[i] = send(req).Code
		}()
	}
	wg.Wait()

	created := 0
	for _, c := range codes {
		switch c {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
		default:
			t.Errorf("unexpected status %d", c)
		}
	}
	if created != 1 {
		t.Errorf("%d books were put on a shelf with limit 1, want 1", created)
	}
}
