package patientapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"

type queryParams struct {
	ID         int    `form:"id"`
	NationalID string `form:"national_id"`
	PassportNo string `form:"passport_no"`
	Phone      string `form:"phone"`
	OrderBy    string `form:"order_by"`
	Page       int    `form:"page"`
	Limit      int    `form:"limit"`
}

func parseFilter(qp queryParams) patientbus.QueryFilter {
	var filter patientbus.QueryFilter
	if qp.ID != 0 {
		filter.ID = &qp.ID
	}

	if qp.NationalID != "" {
		filter.NationalID = &qp.NationalID
	}

	if qp.PassportNo != "" {
		filter.PassportNo = &qp.PassportNo
	}

	if qp.Phone != "" {
		filter.Phone = &qp.Phone
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
