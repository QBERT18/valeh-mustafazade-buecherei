package db

import (
	"gorm.io/gorm"
)

// Person can borrow books.
type Person struct {
	gorm.Model
	Name  string `gorm:"size:255;not null"`
	Loans []Loan `gorm:"foreignKey:PersonID"`
}
