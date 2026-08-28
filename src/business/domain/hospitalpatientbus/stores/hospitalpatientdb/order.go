package hospitalpatientdb

import (
	"fmt"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
)

var orderByFields = map[string]string{
	hospitalpatientbus.OrderByID:           "id",
	hospitalpatientbus.OrderByRegisteredAt: "registered_at",
	hospitalpatientbus.OrderByCreatedAt:    "created_at",
}

func orderByClause(orderBy order.By) (string, error) {
	by, exists := orderByFields[orderBy.Field]
	if !exists {
		return "", fmt.Errorf("field %q does not exist", orderBy.Field)
	}

	return " ORDER BY " + by + " " + orderBy.Direction, nil
}
