package patientapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"

var orderByFields = map[string]string{
	"id":         patientbus.OrderByID,
	"created_at": patientbus.OrderByCreatedAt,
}
