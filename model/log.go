package model

import (
	"gorm.io/gorm"
)

type Log struct {
	gorm.Model
	Method string
	Url    string
	Body   string
	Header string
}
