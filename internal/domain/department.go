package domain

import "time"

type Department struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:255;not null"`
	ParentID  *uint     `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt time.Time `gorm:"not null"`

	Parent    *Department  `gorm:"foreignKey:ParentID;references:ID"`
	Children  []Department `gorm:"foreignKey:ParentID;references:ID"`
	Employees []Employee   `gorm:"foreignKey:DepartmentID;references:ID"`
}

func (Department) TableName() string {
	return "departments"
}
