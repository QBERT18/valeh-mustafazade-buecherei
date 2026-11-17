package api

import (
	"errors"
	"net/http"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type bookInput struct {
	Titel   string `binding:"required"`
	Autor   string
	ISBN    string
	Jahr    datatypes.Date
	ShelfID *uint // nil = not on a shelf

	Praesenzbestand bool
}

// checkShelfHasSpace locks the shelf row until the transaction ends, so two
// requests can't both take the last spot. Call it only when a book is moving
// onto this shelf (it isn't counted yet).
func checkShelfHasSpace(tx *gorm.DB, shelfID uint) error {
	var shelf db.Shelf
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&shelf, shelfID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errShelfNotFound
	}
	if err != nil {
		return err
	}

	var count int64
	if err := tx.Model(&db.Book{}).Where("shelf_id = ?", shelfID).Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(shelf.Limit) {
		return errShelfFull
	}
	return nil
}

func (h *Handler) createBook(c *gin.Context) {
	var in bookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	book := db.Book{Titel: in.Titel, Autor: in.Autor, ISBN: in.ISBN, Jahr: in.Jahr, ShelfID: in.ShelfID,
		Praesenzbestand: in.Praesenzbestand}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if in.ShelfID != nil {
			if err := checkShelfHasSpace(tx, *in.ShelfID); err != nil {
				return err
			}
		}
		return tx.Create(&book).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, book)
}

func (h *Handler) listBooks(c *gin.Context) {
	var books []db.Book
	if err := h.DB.Find(&books).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, books)
}

func (h *Handler) getBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var book db.Book
	if err := h.DB.Preload("Shelf").First(&book, id).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) updateBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in bookInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var book db.Book
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the book so it can't be lent while it becomes Präsenzbestand.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id).Error; err != nil {
			return err
		}
		if in.Praesenzbestand && !book.Praesenzbestand {
			lent, err := hasOpenLoan(tx, id)
			if err != nil {
				return err
			}
			if lent {
				return errBookOnLoan
			}
		}
		movesShelf := in.ShelfID != nil && (book.ShelfID == nil || *book.ShelfID != *in.ShelfID)
		if movesShelf {
			if err := checkShelfHasSpace(tx, *in.ShelfID); err != nil {
				return err
			}
		}

		book.Titel = in.Titel
		book.Autor = in.Autor
		book.ISBN = in.ISBN
		book.Jahr = in.Jahr
		book.ShelfID = in.ShelfID
		book.Praesenzbestand = in.Praesenzbestand
		return tx.Save(&book).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) deleteBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var book db.Book
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id).Error; err != nil {
			return err
		}
		lent, err := hasOpenLoan(tx, id)
		if err != nil {
			return err
		}
		if lent {
			return errBookOnLoan
		}
		return tx.Delete(&book).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
