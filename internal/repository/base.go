package repository

import "gorm.io/gorm"

type baseRepo struct {
	db *gorm.DB
}

func (b baseRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return b.db
}
