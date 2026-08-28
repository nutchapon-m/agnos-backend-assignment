package userdb

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
)

type user struct {
	ID        int        `db:"id"`
	Username  string     `db:"username"`
	Password  string     `db:"password"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

func toDBUser(u userbus.User) user {
	return user{
		ID:        u.ID,
		Username:  u.Username,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func toUser(u user) userbus.User {
	return userbus.User{
		ID:        u.ID,
		Username:  u.Username,
		Password:  u.Password,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
