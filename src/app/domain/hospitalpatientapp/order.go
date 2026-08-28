package hospitalpatientapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"

var orderByFields = map[string]string{
	"id":            hospitalpatientbus.OrderByID,
	"registered_at": hospitalpatientbus.OrderByRegisteredAt,
	"created_at":    hospitalpatientbus.OrderByCreatedAt,
}
