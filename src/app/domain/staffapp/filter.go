package staffapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"

type queryParams struct {
	ID      int    `form:"id"`
	UserID  int    `form:"user_id"`
	OrderBy string `form:"order_by"`
	Page    int    `form:"page"`
	Limit   int    `form:"limit"`
}

func parseFilter(qp queryParams) staffbus.QueryFilter {
	var filter staffbus.QueryFilter
	if qp.ID != 0 {
		filter.ID = &qp.ID
	}

	if qp.UserID != 0 {
		filter.UserID = &qp.UserID
	}

	if qp.OrderBy != "" {
		filter.OrderBy = &qp.OrderBy
	}

	if qp.Page != 0 {
		filter.Page = &qp.Page
	}

	if qp.Limit != 0 {
		filter.Limit = &qp.Limit
	}
	return filter
}
