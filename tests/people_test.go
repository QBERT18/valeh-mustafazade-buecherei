package tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
)

func TestCreatePerson(t *testing.T) {
	reset(t)

	w := call(t, "POST", "/people", gin.H{"Name": "Anna"}, http.StatusCreated)
	person := decode[db.Person](t, w)
	if person.ID == 0 || person.Name != "Anna" {
		t.Errorf("unexpected person: %+v", person)
	}
	if !strings.Contains(w.Body.String(), `"Loans":[]`) {
		t.Errorf("new person should have Loans [], got %s", w.Body)
	}

	call(t, "POST", "/people", gin.H{}, http.StatusBadRequest)
	call(t, "POST", "/people", gin.H{"Name": ""}, http.StatusBadRequest)
}

func TestListPeople(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	createPerson(t, "Ben")
	lend(t, createBook(t, "Faust", nil).ID, anna.ID)

	people := decode[[]db.Person](t, call(t, "GET", "/people", nil, http.StatusOK))
	if len(people) != 2 {
		t.Fatalf("got %d people, want 2", len(people))
	}
	for _, p := range people {
		want := map[string]int{"Anna": 1, "Ben": 0}[p.Name]
		if len(p.Loans) != want {
			t.Errorf("%s has %d loans in the list, want %d", p.Name, len(p.Loans), want)
		}
	}
}

func TestGetPerson(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	book := createBook(t, "Faust", nil)
	lend(t, book.ID, anna.ID)

	got := decode[db.Person](t, call(t, "GET", fmt.Sprintf("/people/%d", anna.ID), nil, http.StatusOK))
	if got.Name != "Anna" || len(got.Loans) != 1 || got.Loans[0].BookID != book.ID {
		t.Errorf("unexpected person (loans should be included): %+v", got)
	}
	if b := got.Loans[0].Book; b == nil || b.Titel != "Faust" {
		t.Errorf("loan should include its book, got %+v", b)
	}

	call(t, "GET", "/people/999", nil, http.StatusNotFound)
	call(t, "GET", "/people/abc", nil, http.StatusBadRequest)
}

func TestUpdatePerson(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	path := fmt.Sprintf("/people/%d", anna.ID)

	got := decode[db.Person](t, call(t, "PUT", path, gin.H{"Name": "Anna Schmidt"}, http.StatusOK))
	if got.Name != "Anna Schmidt" {
		t.Errorf("Name = %q", got.Name)
	}
	if got := decode[db.Person](t, call(t, "GET", path, nil, http.StatusOK)); got.Name != "Anna Schmidt" {
		t.Errorf("update not saved, Name = %q", got.Name)
	}

	call(t, "PUT", path, gin.H{}, http.StatusBadRequest)
	call(t, "PUT", "/people/999", gin.H{"Name": "X"}, http.StatusNotFound)
}

func TestDeletePerson(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	path := fmt.Sprintf("/people/%d", anna.ID)

	t.Run("refused while a book is borrowed", func(t *testing.T) {
		book := createBook(t, "Faust", nil)
		lend(t, book.ID, anna.ID)
		w := call(t, "DELETE", path, nil, http.StatusConflict)
		if msg := errorMessage(t, w); msg != "person still has borrowed books" {
			t.Errorf("error = %q", msg)
		}
		call(t, "POST", fmt.Sprintf("/books/%d/return", book.ID), nil, http.StatusOK)
	})

	call(t, "DELETE", path, nil, http.StatusNoContent)
	call(t, "GET", path, nil, http.StatusNotFound)
	call(t, "DELETE", path, nil, http.StatusNotFound)

	// The returned loan stays as history.
	if loans := decode[[]db.Loan](t, call(t, "GET", "/loans", nil, http.StatusOK)); len(loans) != 1 {
		t.Errorf("got %d loans after deleting the person, want 1", len(loans))
	}
}
