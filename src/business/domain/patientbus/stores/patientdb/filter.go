package patientdb

import (
	"bytes"
	"strings"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
)

func (s *Store) applyFilters(filter patientbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string
	// Add more filters as needed
	if filter.ID != nil {
		data["id"] = filter.ID
		wc = append(wc, "id = :id")
	}

	if filter.NationalID != nil {
		data["national_id"] = filter.NationalID
		wc = append(wc, "national_id = :national_id")
	}

	if filter.PassportNo != nil {
		data["passport_no"] = filter.PassportNo
		wc = append(wc, "passport_no = :passport_no")
	}

	// The table keeps a Thai and an English spelling of every name part, while
	// the filter carries one value. A name matches when either spelling does.
	if filter.FirstName != nil {
		data["first_name"] = filter.FirstName
		wc = append(wc, "(lower(first_name_th) = lower(:first_name) OR lower(first_name_en) = lower(:first_name))")
	}

	if filter.MiddleName != nil {
		data["middle_name"] = filter.MiddleName
		wc = append(wc, "(lower(middle_name_th) = lower(:middle_name) OR lower(middle_name_en) = lower(:middle_name))")
	}

	if filter.LastName != nil {
		data["last_name"] = filter.LastName
		wc = append(wc, "(lower(last_name_th) = lower(:last_name) OR lower(last_name_en) = lower(:last_name))")
	}

	if filter.DateOfBirth != nil {
		data["date_of_birth"] = filter.DateOfBirth
		// The column is a date, the filter a time.Time. Cast so that a value
		// carrying a clock component still compares as the calendar date.
		wc = append(wc, "date_of_birth = (:date_of_birth)::date")
	}

	if filter.Phone != nil {
		data["phone"] = filter.Phone
		wc = append(wc, "phone = :phone")
	}

	if filter.Email != nil {
		data["email"] = filter.Email
		wc = append(wc, "lower(email) = lower(:email)")
	}

	// Default where clause to exclude deleted records
	wc = append(wc, "deleted_at IS NULL")

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
