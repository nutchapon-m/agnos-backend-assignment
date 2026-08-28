package hospitalstaffdb

import (
	"bytes"
	"strings"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
)

func (s *Store) applyFilters(filter hospitalstaffbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string
	// Add more filters as needed
	if filter.ID != nil {
		data["id"] = filter.ID
		wc = append(wc, "id = :id")
	}

	if filter.HospitalID != nil {
		data["hospital_id"] = filter.HospitalID
		wc = append(wc, "hospital_id = :hospital_id")
	}

	if filter.StaffID != nil {
		data["staff_id"] = filter.StaffID
		wc = append(wc, "staff_id = :staff_id")
	}

	if filter.Role != nil {
		data["role"] = filter.Role
		wc = append(wc, "role = :role")
	}

	if filter.IsPrimary != nil {
		data["is_primary"] = filter.IsPrimary
		wc = append(wc, "is_primary = :is_primary")
	}

	// The table has no deleted_at column, an assignment is instead closed by
	// setting effective_to.
	if filter.Active != nil {
		if *filter.Active {
			wc = append(wc, "effective_to IS NULL")
		} else {
			wc = append(wc, "effective_to IS NOT NULL")
		}
	}

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
