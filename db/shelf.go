package db

import (
	"gorm.io/gorm"
)

type Shelf struct {
	gorm.Model
	ShelfName string `gorm:"size:255;not null"`
	Limit     int    `gorm:"not null;check:chk_shelves_limit,\"limit\" > 0"`
	Books     []Book `gorm:"foreignKey:ShelfID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}
