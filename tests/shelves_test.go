package tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
)

func TestCreateShelf(t *testing.T) {
	reset(t)

	w := call(t, "POST", "/shelves", gin.H{"ShelfName": "A1", "Limit": 3}, http.StatusCreated)
	shelf := decode[db.Shelf](t, w)
	if shelf.ID == 0 || shelf.ShelfName != "A1" || shelf.Limit != 3 {
		t.Errorf("unexpected shelf: %+v", shelf)
	}
	if !strings.Contains(w.Body.String(), `"Books":[]`) {
		t.Errorf("new shelf should have Books [], got %s", w.Body)
	}

	invalid := map[string]gin.H{
		"missing name":   {"Limit": 3},
		"missing limit":  {"ShelfName": "A1"},
		"limit zero":     {"ShelfName": "A1", "Limit": 0},
		"limit negative": {"ShelfName": "A1", "Limit": -1},
	}
	for name, body := range invalid {
		t.Run(name, func(t *testing.T) {
			call(t, "POST", "/shelves", body, http.StatusBadRequest)
		})
	}
}

func TestListShelves(t *testing.T) {
	reset(t)
	a := createShelf(t, "A", 3)
	createShelf(t, "B", 3)
	createBook(t, "Faust", &a.ID)

	shelves := decode[[]db.Shelf](t, call(t, "GET", "/shelves", nil, http.StatusOK))
	if len(shelves) != 2 {
		t.Fatalf("got %d shelves, want 2", len(shelves))
	}
	byName := map[string]db.Shelf{}
	for _, s := range shelves {
		byName[s.ShelfName] = s
	}
	if n := len(byName["A"].Books); n != 1 {
		t.Errorf("shelf A has %d books in the list, want 1", n)
	}
	if b := byName["B"].Books; b == nil || len(b) != 0 {
		t.Errorf("shelf B should have an empty (not null) book list, got %v", b)
	}
}

func TestGetShelf(t *testing.T) {
	reset(t)
	shelf := createShelf(t, "A", 3)
	createBook(t, "Faust", &shelf.ID)

	got := decode[db.Shelf](t, call(t, "GET", fmt.Sprintf("/shelves/%d", shelf.ID), nil, http.StatusOK))
	if got.ShelfName != "A" || len(got.Books) != 1 || got.Books[0].Titel != "Faust" {
		t.Errorf("unexpected shelf: %+v", got)
	}

	call(t, "GET", "/shelves/999", nil, http.StatusNotFound)
	call(t, "GET", "/shelves/abc", nil, http.StatusBadRequest)
}

func TestUpdateShelf(t *testing.T) {
	reset(t)
	shelf := createShelf(t, "A", 3)
	path := fmt.Sprintf("/shelves/%d", shelf.ID)

	got := decode[db.Shelf](t, call(t, "PUT", path, gin.H{"ShelfName": "A-neu", "Limit": 5}, http.StatusOK))
	if got.ShelfName != "A-neu" || got.Limit != 5 {
		t.Errorf("update not applied: %+v", got)
	}

	createBook(t, "Faust", &shelf.ID)
	createBook(t, "Effi Briest", &shelf.ID)
	w := call(t, "PUT", path, gin.H{"ShelfName": "A", "Limit": 1}, http.StatusConflict)
	if msg := errorMessage(t, w); !strings.Contains(msg, "limit") {
		t.Errorf("unexpected error: %q", msg)
	}
	call(t, "PUT", path, gin.H{"ShelfName": "A", "Limit": 2}, http.StatusOK) // exactly full is fine

	call(t, "PUT", path, gin.H{"Limit": 2}, http.StatusBadRequest)
	call(t, "PUT", "/shelves/999", gin.H{"ShelfName": "X", "Limit": 2}, http.StatusNotFound)
}

func TestDeleteShelf(t *testing.T) {
	reset(t)
	shelf := createShelf(t, "A", 3)
	book := createBook(t, "Faust", &shelf.ID)
	path := fmt.Sprintf("/shelves/%d", shelf.ID)

	call(t, "DELETE", path, nil, http.StatusNoContent)
	call(t, "GET", path, nil, http.StatusNotFound)
	call(t, "DELETE", path, nil, http.StatusNotFound)

	// The book stays, but is no longer on a shelf.
	got := decode[db.Book](t, call(t, "GET", fmt.Sprintf("/books/%d", book.ID), nil, http.StatusOK))
	if got.ShelfID != nil {
		t.Errorf("book should have no shelf after the shelf is deleted, got ShelfID %d", *got.ShelfID)
	}
}
