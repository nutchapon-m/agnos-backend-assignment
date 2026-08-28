package hospitalapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"

var orderByFields = map[string]string{
	"id":         hospitalbus.OrderByID,
	"code":       hospitalbus.OrderByCode,
	"created_at": hospitalbus.OrderByCreatedAt,
}
