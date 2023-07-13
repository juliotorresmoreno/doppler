package model

import (
	"time"
)

type User struct {
	Id        uint      `gorm:""            json:"id"`
	Name      string    `gorm:""            json:"name"`
	Lastname  string    `gorm:""            json:"lastname"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Password  string    `gorm:""            json:"-"`
	CreatedAt time.Time `                   json:"-"`
	UpdatedAt time.Time `                   json:"-"`
	DeletedAt time.Time `gorm:"index"       json:"-"`
}
