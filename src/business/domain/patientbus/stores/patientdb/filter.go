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

	if filter.Phone != nil {
		data["phone"] = filter.Phone
		wc = append(wc, "phone = :phone")
	}

	// Default where clause to exclude deleted records
	wc = append(wc, "deleted_at IS NULL")

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
