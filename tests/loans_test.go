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

func TestLendBook(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	ben := createPerson(t, "Ben")
	book := createBook(t, "Faust", nil)
	path := fmt.Sprintf("/books/%d/loan", book.ID)

	before := time.Now()
	loan := lend(t, book.ID, anna.ID)
	if loan.ID == 0 || loan.BookID != book.ID || loan.PersonID != anna.ID {
		t.Errorf("unexpected loan: %+v", loan)
	}
	if loan.LoanedAt.Before(before.Add(-time.Second)) {
		t.Errorf("LoanedAt = %v, want about now", loan.LoanedAt)
	}
	if loan.ReturnedAt != nil {
		t.Errorf("ReturnedAt = %v, want nil", loan.ReturnedAt)
	}

	t.Run("already on loan", func(t *testing.T) {
		w := call(t, "POST", path, gin.H{"PersonID": ben.ID}, http.StatusConflict)
		if msg := errorMessage(t, w); msg != "book is on loan" {
			t.Errorf("error = %q", msg)
		}
	})
	t.Run("Präsenzbestand", func(t *testing.T) {
		lexikon := decode[db.Book](t, call(t, "POST", "/books",
			gin.H{"Titel": "Brockhaus", "Praesenzbestand": true}, http.StatusCreated))
		w := call(t, "POST", fmt.Sprintf("/books/%d/loan", lexikon.ID), gin.H{"PersonID": anna.ID}, http.StatusConflict)
		if msg := errorMessage(t, w); msg != "book is Präsenzbestand and cannot be lent" {
			t.Errorf("error = %q", msg)
		}
	})
	t.Run("unknown person", func(t *testing.T) {
		other := createBook(t, "Effi Briest", nil)
		w := call(t, "POST", fmt.Sprintf("/books/%d/loan", other.ID), gin.H{"PersonID": 999}, http.StatusBadRequest)
		if msg := errorMessage(t, w); msg != "person not found" {
			t.Errorf("error = %q", msg)
		}
	})
	t.Run("unknown book", func(t *testing.T) {
		call(t, "POST", "/books/999/loan", gin.H{"PersonID": anna.ID}, http.StatusNotFound)
	})
	t.Run("missing person", func(t *testing.T) {
		call(t, "POST", path, gin.H{}, http.StatusBadRequest)
	})
}

func TestReturnBook(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	ben := createPerson(t, "Ben")
	book := createBook(t, "Faust", nil)
	path := fmt.Sprintf("/books/%d/return", book.ID)

	t.Run("not on loan", func(t *testing.T) {
		w := call(t, "POST", path, nil, http.StatusConflict)
		if msg := errorMessage(t, w); msg != "book is not on loan" {
			t.Errorf("error = %q", msg)
		}
	})

	loan := lend(t, book.ID, anna.ID)
	returned := decode[db.Loan](t, call(t, "POST", path, nil, http.StatusOK))
	if returned.ID != loan.ID || returned.ReturnedAt == nil {
		t.Errorf("loan not closed: %+v", returned)
	}

	call(t, "POST", path, nil, http.StatusConflict) // already returned

	t.Run("can be lent again after return", func(t *testing.T) {
		again := lend(t, book.ID, ben.ID)
		if again.ID == loan.ID {
			t.Error("a new loan should be created, not the old one reused")
		}
	})
	t.Run("unknown book", func(t *testing.T) {
		call(t, "POST", "/books/999/return", nil, http.StatusNotFound)
	})
}

func TestListLoans(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	book := createBook(t, "Faust", nil)
	lend(t, book.ID, anna.ID)
	call(t, "POST", fmt.Sprintf("/books/%d/return", book.ID), nil, http.StatusOK)
	lend(t, book.ID, anna.ID)

	loans := decode[[]db.Loan](t, call(t, "GET", "/loans", nil, http.StatusOK))
	if len(loans) != 2 {
		t.Fatalf("got %d loans, want 2 (history is kept)", len(loans))
	}
	if loans[0].ReturnedAt == nil || loans[1].ReturnedAt != nil {
		t.Errorf("want first loan returned and second open, got %+v", loans)
	}
}

func TestGetLoan(t *testing.T) {
	reset(t)
	anna := createPerson(t, "Anna")
	book := createBook(t, "Faust", nil)
	loan := lend(t, book.ID, anna.ID)

	got := decode[db.Loan](t, call(t, "GET", fmt.Sprintf("/loans/%d", loan.ID), nil, http.StatusOK))
	if got.Book == nil || got.Book.Titel != "Faust" || got.Person == nil || got.Person.Name != "Anna" {
		t.Errorf("loan should include book and person: %+v", got)
	}

	call(t, "GET", "/loans/999", nil, http.StatusNotFound)
	call(t, "GET", "/loans/abc", nil, http.StatusBadRequest)
}

func TestLoansCannotBeEditedOrDeleted(t *testing.T) {
	reset(t)
	loan := lend(t, createBook(t, "Faust", nil).ID, createPerson(t, "Anna").ID)
	path := fmt.Sprintf("/loans/%d", loan.ID)

	call(t, "PUT", path, gin.H{}, http.StatusNotFound)
	call(t, "DELETE", path, nil, http.StatusNotFound)
	call(t, "POST", "/loans", gin.H{}, http.StatusNotFound)
}

// Two people must not be able to borrow the same book at the same moment.
func TestLendUnderConcurrency(t *testing.T) {
	reset(t)
	book := createBook(t, "Faust", nil)
	const n = 10
	people := make([]db.Person, n)
	for i := range people {
		people[i] = createPerson(t, fmt.Sprintf("Person %d", i))
	}

	codes := make([]int, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, _ := json.Marshal(gin.H{"PersonID": people[i].ID})
			req := httptest.NewRequest("POST", fmt.Sprintf("/books/%d/loan", book.ID), strings.NewReader(string(body)))
			req.Header.Set("Content-Type", "application/json")
			codes[i] = send(req).Code
		}()
	}
	wg.Wait()

	lent := 0
	for _, c := range codes {
		switch c {
		case http.StatusCreated:
			lent++
		case http.StatusConflict:
		default:
			t.Errorf("unexpected status %d", c)
		}
	}
	if lent != 1 {
		t.Errorf("book was lent %d times at once, want 1", lent)
	}
}
