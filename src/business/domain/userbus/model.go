package userbus

import "time"

type User struct {
	ID        int
	Username  string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewUser struct {
	Username string
	Password string
}
