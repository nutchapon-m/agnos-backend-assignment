package hospitalpatientapp

import "github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"

type queryParams struct {
	ID         int    `form:"id"`
	HospitalID int    `form:"hospital_id"`
	PatientID  int    `form:"patient_id"`
	HN         string `form:"hn"`
	Status     string `form:"status"`
	OrderBy    string `form:"order_by"`
	Page       int    `form:"page"`
	Limit      int    `form:"limit"`
}

func parseFilter(qp queryParams) hospitalpatientbus.QueryFilter {
	var filter hospitalpatientbus.QueryFilter
	if qp.ID != 0 {
		filter.ID = &qp.ID
	}

	if qp.HospitalID != 0 {
		filter.HospitalID = &qp.HospitalID
	}

	if qp.PatientID != 0 {
		filter.PatientID = &qp.PatientID
	}

	if qp.HN != "" {
		filter.HN = &qp.HN
	}

	if qp.Status != "" {
		filter.Status = &qp.Status
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
