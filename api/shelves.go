package api

import (
	"net/http"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type shelfInput struct {
	ShelfName string `binding:"required"`
	Limit     int    `binding:"required,gt=0"`
}

func (h *Handler) createShelf(c *gin.Context) {
	var in shelfInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shelf := db.Shelf{ShelfName: in.ShelfName, Limit: in.Limit, Books: []db.Book{}} // [] instead of null in the response
	if err := h.DB.Create(&shelf).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, shelf)
}

func (h *Handler) listShelves(c *gin.Context) {
	q, err := searchAndSort(c, h.DB, []string{"shelf_name"},
		map[string]string{"name": "shelf_name", "limit": "limit"})
	if err != nil {
		writeError(c, err)
		return
	}
	var shelves []db.Shelf
	if err := q.Preload("Books").Find(&shelves).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, shelves)
}

func (h *Handler) getShelf(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var shelf db.Shelf
	if err := h.DB.Preload("Books").First(&shelf, id).Error; err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, shelf)
}

func (h *Handler) updateShelf(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in shelfInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var shelf db.Shelf
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the shelf so no book can be added while we check the new limit.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&shelf, id).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&db.Book{}).Where("shelf_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > int64(in.Limit) {
			return errLimitBelowCount
		}

		shelf.ShelfName = in.ShelfName
		shelf.Limit = in.Limit
		if err := tx.Save(&shelf).Error; err != nil {
			return err
		}
		return tx.Preload("Books").First(&shelf, id).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, shelf)
}

func (h *Handler) deleteShelf(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var shelf db.Shelf
		if err := tx.First(&shelf, id).Error; err != nil {
			return err
		}
		// Shelves are soft-deleted, so the FK's ON DELETE SET NULL never fires:
		// take the books off the shelf explicitly.
		if err := tx.Model(&db.Book{}).Where("shelf_id = ?", id).Update("shelf_id", nil).Error; err != nil {
			return err
		}
		return tx.Delete(&shelf).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
