package hospitalapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"

type queryParams struct {
	ID           int    `form:"id"`
	Code         string `form:"code"`
	ProvinceCode string `form:"province_code"`
	IsActive     *bool  `form:"is_active"`
	OrderBy      string `form:"order_by"`
	Page         int    `form:"page"`
	Limit        int    `form:"limit"`
}

func parseFilter(qp queryParams) hospitalbus.QueryFilter {
	var filter hospitalbus.QueryFilter
	if qp.ID != 0 {
		filter.ID = &qp.ID
	}

	if qp.Code != "" {
		filter.Code = &qp.Code
	}

	if qp.ProvinceCode != "" {
		filter.ProvinceCode = &qp.ProvinceCode
	}

	if qp.IsActive != nil {
		filter.IsActive = qp.IsActive
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
