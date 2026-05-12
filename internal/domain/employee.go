package domain

import "time"

type Employee struct {
	ID           uint      `gorm:"primaryKey"`
	DepartmentID uint      `gorm:"not null;index;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	FullName     string    `gorm:"size:255;not null"`
	Position     string    `gorm:"size:255;not null"`
	HiredAt      time.Time `gorm:"type:date;not null"`
	CreatedAt    time.Time `gorm:"not null"`

	Department Department `gorm:"foreignKey:DepartmentID;references:ID"`
}

func (Employee) TableName() string {
	return "employees"
}
