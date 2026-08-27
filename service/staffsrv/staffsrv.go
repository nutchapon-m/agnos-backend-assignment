package staffsrv

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, staff Staff) error
	Query(ctx context.Context) ([]Staff, error)
	GetByID(ctx context.Context, id uuid.UUID) (Staff, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
