package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/QBERT18/valeh-mustafazade-buecherei/storage"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	DB      *gorm.DB
	Storage *storage.Storage
}

func Register(r gin.IRouter, g *gorm.DB, st *storage.Storage) {
	h := &Handler{DB: g, Storage: st}

	shelves := r.Group("/shelves")
	shelves.POST("", h.createShelf)
	shelves.GET("", h.listShelves)
	shelves.GET("/:id", h.getShelf)
	shelves.PUT("/:id", h.updateShelf)
	shelves.DELETE("/:id", h.deleteShelf)

	books := r.Group("/books")
	books.POST("", h.createBook)
	books.GET("", h.listBooks)
	books.GET("/:id", h.getBook)
	books.PUT("/:id", h.updateBook)
	books.DELETE("/:id", h.deleteBook)
	books.PUT("/:id/file", h.uploadBookFile)
	books.GET("/:id/file", h.downloadBookFile)
	books.DELETE("/:id/file", h.deleteBookFile)
	books.POST("/:id/loan", h.lendBook)
	books.POST("/:id/return", h.returnBook)

	people := r.Group("/people")
	people.POST("", h.createPerson)
	people.GET("", h.listPeople)
	people.GET("/:id", h.getPerson)
	people.PUT("/:id", h.updatePerson)
	people.DELETE("/:id", h.deletePerson)

	loans := r.Group("/loans")
	loans.GET("", h.listLoans)
	loans.GET("/:id", h.getLoan)
}

var (
	errShelfNotFound   = errors.New("shelf not found")
	errShelfFull       = errors.New("shelf is full")
	errLimitBelowCount = errors.New("limit is lower than the number of books on the shelf")
	errNoFile          = errors.New("book has no file")
	errPersonNotFound  = errors.New("person not found")
	errPersonHasLoans  = errors.New("person still has borrowed books")
	errPraesenzbestand = errors.New("book is Präsenzbestand and cannot be lent")
	errBookOnLoan      = errors.New("book is on loan")
	errBookNotOnLoan   = errors.New("book is not on loan")
)

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return uint(id), true
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, errNoFile):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, errShelfNotFound), errors.Is(err, errPersonNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, errShelfFull), errors.Is(err, errLimitBelowCount),
		errors.Is(err, errPersonHasLoans), errors.Is(err, errPraesenzbestand),
		errors.Is(err, errBookOnLoan), errors.Is(err, errBookNotOnLoan):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		log.Printf("internal error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
