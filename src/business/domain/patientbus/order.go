package patientbus

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"

var DefaultOrderBy = order.NewBy(OrderByID, order.ASC)

// Set of fields that the results can be ordered by.
const (
	OrderByID        = "a"
	OrderByCreatedAt = "b"
)
