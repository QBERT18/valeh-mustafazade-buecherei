package api

import (
	"context"
	"errors"
	"log"
	"mime"
	"net/http"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/QBERT18/valeh-mustafazade-buecherei/storage"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxFileSize = 50 << 20

var allowedTypes = map[string]string{
	"application/pdf":      ".pdf",
	"application/epub+zip": ".epub",
}

func (h *Handler) uploadBookFile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize+1<<20)
	fh, err := c.FormFile("file")
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || (err == nil && fh.Size > maxFileSize) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file is larger than 50 MB"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "form field \"file\" is missing"})
		return
	}

	f, err := fh.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer f.Close()

	mt, err := mimetype.DetectReader(f)
	if err != nil {
		writeError(c, err)
		return
	}
	ext, allowed := allowedTypes[mt.String()]
	if !allowed {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "only PDF and EPUB files are allowed, got " + mt.String()})
		return
	}
	if _, err := f.Seek(0, 0); err != nil {
		writeError(c, err)
		return
	}

	var book db.Book
	if err := h.DB.First(&book, id).Error; err != nil {
		writeError(c, err)
		return
	}

	ctx := c.Request.Context()
	key := storage.BookFileKey(id, ext)
	if _, err := h.Storage.Client.PutObject(ctx, h.Storage.Bucket, key, f, fh.Size,
		minio.PutObjectOptions{ContentType: mt.String()}); err != nil {
		writeError(c, err)
		return
	}

	var oldKey *string
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id).Error; err != nil {
			return err
		}
		oldKey = book.FileKey
		book.FileKey = &key
		book.FileName = fh.Filename
		book.FileContentType = mt.String()
		book.FileSize = fh.Size
		return tx.Save(&book).Error
	})
	if err != nil {
		h.removeObject(key)
		writeError(c, err)
		return
	}

	if oldKey != nil {
		h.removeObject(*oldKey)
	}
	c.JSON(http.StatusOK, book)
}

func (h *Handler) downloadBookFile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var book db.Book
	if err := h.DB.First(&book, id).Error; err != nil {
		writeError(c, err)
		return
	}
	if book.FileKey == nil {
		writeError(c, errNoFile)
		return
	}

	obj, err := h.Storage.Client.GetObject(c.Request.Context(), h.Storage.Bucket, *book.FileKey, minio.GetObjectOptions{})
	if err != nil {
		writeError(c, err)
		return
	}
	defer obj.Close()
	// GetObject is lazy; Stat surfaces a missing object before we send headers.
	info, err := obj.Stat()
	if err != nil {
		writeError(c, err)
		return
	}

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": book.FileName})
	c.DataFromReader(http.StatusOK, info.Size, book.FileContentType, obj,
		map[string]string{"Content-Disposition": disposition})
}

func (h *Handler) deleteBookFile(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var oldKey *string
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var book db.Book
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, id).Error; err != nil {
			return err
		}
		if book.FileKey == nil {
			return errNoFile
		}
		oldKey = book.FileKey
		return tx.Model(&book).Updates(map[string]any{
			"file_key": nil, "file_name": "", "file_content_type": "", "file_size": 0,
		}).Error
	})
	if err != nil {
		writeError(c, err)
		return
	}

	h.removeObject(*oldKey)
	c.Status(http.StatusNoContent)
}

func (h *Handler) removeObject(key string) {
	if err := h.Storage.Client.RemoveObject(context.Background(), h.Storage.Bucket, key, minio.RemoveObjectOptions{}); err != nil {
		log.Printf("could not remove orphaned object %q: %v", key, err)
	}
}
