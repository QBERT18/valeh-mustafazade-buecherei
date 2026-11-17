package db

import (
	"time"

	"gorm.io/gorm"
)

// Loan records that a book was lent to a person. Returning the book sets
// ReturnedAt, so past loans stay as history.
type Loan struct {
	gorm.Model
	// The partial unique index allows at most one open loan per book.
	BookID uint  `gorm:"not null;index;uniqueIndex:idx_loans_open_book,where:returned_at IS NULL AND deleted_at IS NULL"`
	Book   *Book `gorm:"foreignKey:BookID"`

	PersonID uint    `gorm:"not null;index"`
	Person   *Person `gorm:"foreignKey:PersonID"`

	LoanedAt   time.Time  `gorm:"not null"`
	ReturnedAt *time.Time // nil = the book is still on loan
}
