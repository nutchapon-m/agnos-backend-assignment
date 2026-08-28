package userapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"

type queryParams struct {
	ID      int    `form:"id"`
	OrderBy string `form:"order_by"`
	Page    int    `form:"page"`
	Limit   int    `form:"limit"`
}

func parseFilter(qp queryParams) userbus.QueryFilter {
	var filter userbus.QueryFilter
	if qp.ID != 0 {
		filter.ID = &qp.ID
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
