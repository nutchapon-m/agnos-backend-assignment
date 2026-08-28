package hospitalstaffapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"

type queryParams struct {
	ID         int    `form:"id"`
	HospitalID int    `form:"hospital_id"`
	StaffID    int    `form:"staff_id"`
	Role       string `form:"role"`
	IsPrimary  *bool  `form:"is_primary"`
	Active     *bool  `form:"active"`
	OrderBy    string `form:"order_by"`
	Page       int    `form:"page"`
	Limit      int    `form:"limit"`
}

func parseFilter(qp queryParams) hospitalstaffbus.QueryFilter {
	var filter hospitalstaffbus.QueryFilter
	if qp.ID != 0 {
		filter.ID = &qp.ID
	}

	if qp.HospitalID != 0 {
		filter.HospitalID = &qp.HospitalID
	}

	if qp.StaffID != 0 {
		filter.StaffID = &qp.StaffID
	}

	if qp.Role != "" {
		filter.Role = &qp.Role
	}

	if qp.IsPrimary != nil {
		filter.IsPrimary = qp.IsPrimary
	}

	if qp.Active != nil {
		filter.Active = qp.Active
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
