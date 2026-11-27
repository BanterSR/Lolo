package db

import (
	"errors"

	"gorm.io/gorm"
)

type OFQuickCheck struct {
	UID             string `gorm:"primarykey;unique;index"`
	GateToken       string
	LastPackageName string
}

func OrCreateOFQuickCheck(uid string) (*OFQuickCheck, error) {
	accountCreateMu.Lock()
	defer accountCreateMu.Unlock()

	q := &OFQuickCheck{
		UID: uid,
	}
	err := db.Where("uid = ?", uid).First(q).Error
	if err == nil {
		return q, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err = db.Create(q).Error; err != nil {
		return nil, err
	}
	return q, nil
}

func UpOFQuickCheck(q *OFQuickCheck) error {
	return db.Save(q).Error
}
