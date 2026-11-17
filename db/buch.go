package db

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Book struct {
	gorm.Model
	Titel string
	Autor string
	ISBN  string
	Jahr  datatypes.Date

	// Präsenzbestand (e.g. encyclopedias): may be read in the library, but not lent.
	Praesenzbestand bool `gorm:"not null;default:false"`

	ShelfID *uint  `gorm:"index"` // nil = book is not on a shelf
	Shelf   *Shelf `gorm:"foreignKey:ShelfID"`

	// The book's file lives in object storage; the row only references it.
	FileKey         *string `gorm:"size:512;uniqueIndex" json:"-"` // nil = no file uploaded
	FileName        string  // original name, sent back on download
	FileContentType string
	FileSize        int64
}
