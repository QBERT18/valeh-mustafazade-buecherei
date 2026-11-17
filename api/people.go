package api

import (
	"net/http"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type personInput struct {
	Name string `binding:"required"`
}

func (h *Handler) createPerson(c *gin.Context) {
	var in personInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	person := db.Person{Name: in.Name, Loans: []db.Loan{}} // [] instead of null in the response
	if err := h.DB.Create(&person).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, person)
}

func (h *Handler) listPeople(c *gin.Context) {
	var people []db.Person
	if err := h.DB.Preload("Loans").Find(&people).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, people)
}

func (h *Handler) getPerson(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var person db.Person
	if err := h.DB.Preload("Loans").First(&person, id).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, person)
}

func (h *Handler) updatePerson(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in personInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var person db.Person
	if err := h.DB.First(&person, id).Error; err != nil {
		writeError(c, err)
		return
	}
	person.Name = in.Name
	if err := h.DB.Save(&person).Error; err != nil {
		writeError(c, err)
		return
	}
	if err := h.DB.Preload("Loans").First(&person, id).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, person)
}

// deletePerson refuses while the person still has a book; their returned
// loans stay as history.
func (h *Handler) deletePerson(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var person db.Person
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&person, id).Error; err != nil {
			return err
		}
		var open int64
		if err := tx.Model(&db.Loan{}).Where("person_id = ? AND returned_at IS NULL", id).Count(&open).Error; err != nil {
			return err
		}
		if open > 0 {
			return errPersonHasLoans
		}
		return tx.Delete(&person).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
