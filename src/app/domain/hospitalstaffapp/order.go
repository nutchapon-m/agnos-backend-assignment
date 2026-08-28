package hospitalstaffapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"

var orderByFields = map[string]string{
	"id":             hospitalstaffbus.OrderByID,
	"effective_from": hospitalstaffbus.OrderByEffectiveFrom,
	"created_at":     hospitalstaffbus.OrderByCreatedAt,
}
