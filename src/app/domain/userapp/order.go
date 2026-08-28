package userapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"

var orderByFields = map[string]string{
	"id":         userbus.OrderByID,
	"created_at": userbus.OrderByCreatedAt,
}
