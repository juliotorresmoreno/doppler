package model

import (
	"time"

	"gorm.io/gorm"
)

type Log struct {
	gorm.Model
	Method    string
	Url       string
	Body      string
	Header    string
	CreatedAt time.Time `gorm:"autoUpdateTime:true"`
	UpdatedAt time.Time `gorm:"autoUpdateTime:true"`
	DeletedAt time.Time `gorm:"autoUpdateTime:true"`
}
