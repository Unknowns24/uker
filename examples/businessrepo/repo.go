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
	db           *gorm.DB
	cursorSecret []byte
	cursorTTL    time.Duration
}

type BusinessMember struct {
	ID         string    `gorm:"column:id;primaryKey"`
	BusinessID string    `gorm:"column:business_id"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (BusinessMember) TableName() string {
	return "business_members"
}

func NewBusinessRepo(db *gorm.DB, secret []byte, ttl time.Duration) (*BusinessRepo, error) {
	if db == nil {
		return nil, errNilDB
	}
	if len(secret) == 0 {
		return nil, errors.New("businessrepo: missing cursor secret")
	}
	if ttl < 0 {
		ttl = 0
	}
	return &BusinessRepo{db: db, cursorSecret: append([]byte(nil), secret...), cursorTTL: ttl}, nil
}

func (r *BusinessRepo) ListMembers(businessID string, raw url.Values) (*pagination.PagingResponse[BusinessMember], error) {
	if r == nil || r.db == nil {
		return nil, errNilDB
	}

	params, err := pagination.ParseWithSecurity(raw, r.cursorSecret, r.cursorTTL)
	if err != nil {
		return nil, err
	}

	base := r.db.Model(&BusinessMember{}).Where("business_id = ?", businessID)
	countParams := params
	countParams.Cursor = nil
	countParams.RawCursor = ""
	countParams.Limit = 0

	countQuery, err := pagination.Apply(base, countParams)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	query, err := pagination.Apply(base, params)
	if err != nil {
		return nil, err
	}

	var results []BusinessMember
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}

	page, err := pagination.BuildPageSigned[BusinessMember](params, results, limit, total, nil, r.cursorSecret)
	if err != nil {
		return nil, err
	}

	return &page, nil
}
