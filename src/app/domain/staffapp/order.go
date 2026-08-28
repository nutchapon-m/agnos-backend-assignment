package staffapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"

var orderByFields = map[string]string{
	"id":         staffbus.OrderByID,
	"created_at": staffbus.OrderByCreatedAt,
}
