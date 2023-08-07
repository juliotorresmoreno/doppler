package model

import "time"

type Server struct {
	Id          uint      `gorm:""                   json:"id"`
	Name        string    `gorm:""                   json:"name"`
	IpAddress   string    `gorm:""                   json:"ip_address"`
	Description string    `gorm:"text"               json:"description"`
	OwnerID     uint      `gorm:""                   json:"-"`
	Owner       User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"      json:"owner"`
	CreatedAt   time.Time `                          json:"-"`
	UpdatedAt   time.Time `                          json:"-"`
	DeletedAt   time.Time `gorm:"index"              json:"-"`
}

func (s Server) TableName() string {
	return "servers"
}
