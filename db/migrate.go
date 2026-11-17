package db

import "gorm.io/gorm"

// Migrate creates or updates the tables for all models.
func Migrate(g *gorm.DB) error {
	return g.AutoMigrate(&Shelf{}, &Book{}, &Person{}, &Loan{})
}
