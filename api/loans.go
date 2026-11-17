package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Loans are history: they are created by lending and closed by returning,
// but never edited or deleted.

type loanInput struct {
	PersonID uint `binding:"required"`
}

// hasOpenLoan reports whether the book is currently lent out.
func hasOpenLoan(tx *gorm.DB, bookID uint) (bool, error) {
	var open int64
	err := tx.Model(&db.Loan{}).Where("book_id = ? AND returned_at IS NULL", bookID).Count(&open).Error
	return open > 0, err
}

func (h *Handler) lendBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in loanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var loan db.Loan
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the book so it can't be lent twice or become Präsenzbestand meanwhile.
		var book db.Book
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id).Error; err != nil {
			return err
		}
		if book.Praesenzbestand {
			return errPraesenzbestand
		}

		var person db.Person
		err := tx.First(&person, in.PersonID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errPersonNotFound
		}
		if err != nil {
			return err
		}

		lent, err := hasOpenLoan(tx, id)
		if err != nil {
			return err
		}
		if lent {
			return errBookOnLoan
		}

		loan = db.Loan{BookID: id, PersonID: in.PersonID, LoanedAt: time.Now()}
		return tx.Create(&loan).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, loan)
}

func (h *Handler) returnBook(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var loan db.Loan
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var book db.Book
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id).Error; err != nil {
			return err
		}
		err := tx.Where("book_id = ? AND returned_at IS NULL", id).First(&loan).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errBookNotOnLoan
		}
		if err != nil {
			return err
		}

		now := time.Now()
		loan.ReturnedAt = &now
		return tx.Save(&loan).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, loan)
}

func (h *Handler) listLoans(c *gin.Context) {
	var loans []db.Loan
	if err := h.DB.Order("id").Find(&loans).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, loans)
}

func (h *Handler) getLoan(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var loan db.Loan
	if err := h.DB.Preload("Book").Preload("Person").First(&loan, id).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, loan)
}
