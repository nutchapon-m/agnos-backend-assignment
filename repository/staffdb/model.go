package staffdb

import "time"

type staff struct {
	ID         string     `db:"id"`
	FirstName  string     `db:"first_name"`
	LastName   string     `db:"last_name"`
	HospitalID string     `db:"hospital_id"`
	CreatedBy  string     `db:"created_by"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedBy  string     `db:"updated_by"`
	UpdatedAt  time.Time  `db:"updated_at"`
	DeletedAt  *time.Time `db:"deleted_at"`
}
