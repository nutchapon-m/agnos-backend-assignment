package userapp

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
)

var (
	isoLayout = time.RFC3339
)

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toAppUser(u userbus.User) User {
	return User{
		ID:        u.ID,
		Username:  u.Username,
		CreatedAt: u.CreatedAt.Format(isoLayout),
		UpdatedAt: u.UpdatedAt.Format(isoLayout),
	}
}

func toAppUsers(users []userbus.User) []User {
	items := make([]User, len(users))
	for i, u := range users {
		items[i] = toAppUser(u)
	}
	return items
}

type NewUser struct {
	Username string `json:"username" binding:"required,min=3,max=255"`
	Password string `json:"password" binding:"required,min=6"`
}

func toNewUser(nu NewUser) userbus.NewUser {
	return userbus.NewUser{
		Username: nu.Username,
		Password: nu.Password,
	}
}
