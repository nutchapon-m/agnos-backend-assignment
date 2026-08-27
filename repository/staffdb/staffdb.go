package staffdb

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/service/staffsrv"
)

type Repository struct {
	db sqlx.ExtContext
}

func NewRepository(db sqlx.ExtContext) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, staff staffsrv.Staff) error {
	return nil
}

func (r *Repository) Query(ctx context.Context) ([]staffsrv.Staff, error) {
	return nil, nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (staffsrv.Staff, error) {
	return staffsrv.Staff{}, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
