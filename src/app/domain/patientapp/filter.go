package patientapp

import (
	"time"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
)

type queryParams struct {
	ID          int        `form:"id"`
	NationalID  string     `form:"national_id"`
	PassportNo  string     `form:"passport_no"`
	FirstName   string     `form:"first_name"`
	MiddleName  string     `form:"middle_name"`
	LastName    string     `form:"last_name"`
	Phone       string     `form:"phone_number"`
	Email       string     `form:"email"`
	DateOfBirth *time.Time `form:"date_of_birth"`
	OrderBy     string     `form:"order_by"`
	Page        int        `form:"page"`
	Limit       int        `form:"limit"`
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

	if qp.FirstName != "" {
		filter.FirstName = &qp.FirstName
	}

	if qp.MiddleName != "" {
		filter.MiddleName = &qp.MiddleName
	}

	if qp.LastName != "" {
		filter.LastName = &qp.LastName
	}

	if qp.Phone != "" {
		filter.Phone = &qp.Phone
	}

	if qp.Email != "" {
		filter.Email = &qp.Email
	}

	if qp.DateOfBirth != nil {
		filter.DateOfBirth = qp.DateOfBirth
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
