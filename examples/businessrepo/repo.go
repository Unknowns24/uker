package businessrepo

import (
	"errors"
	"net/url"
	"time"

	"github.com/unknowns24/uker/uker/pagination"
	"gorm.io/gorm"
)

var errNilDB = errors.New("businessrepo: nil db")

type BusinessRepo struct {
	db *gorm.DB
}

type BusinessMember struct {
	ID         string    `gorm:"column:id;primaryKey"`
	BusinessID string    `gorm:"column:business_id"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (BusinessMember) TableName() string {
	return "business_members"
}

func NewBusinessRepo(db *gorm.DB) (*BusinessRepo, error) {
	if db == nil {
		return nil, errNilDB
	}
	return &BusinessRepo{db: db}, nil
}

func (r *BusinessRepo) ListMembers(businessID string, raw url.Values) (*pagination.PagingResponse[BusinessMember], error) {
	if r == nil || r.db == nil {
		return nil, errNilDB
	}

	params, err := pagination.Parse(raw)
	if err != nil {
		return nil, err
	}

	base := r.db.Model(&BusinessMember{}).Where("business_id = ?", businessID)
	query, err := pagination.Apply(base, params)
	if err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}

	query = query.Limit(limit + 1)

	var results []BusinessMember
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	page, err := pagination.BuildPage[BusinessMember](params, results, limit, nil)
	if err != nil {
		return nil, err
	}

	return &page, nil
}
