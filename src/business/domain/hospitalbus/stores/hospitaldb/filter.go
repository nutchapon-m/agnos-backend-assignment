package hospitaldb

import (
	"bytes"
	"strings"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
)

func (s *Store) applyFilters(filter hospitalbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string
	// Add more filters as needed
	if filter.ID != nil {
		data["id"] = filter.ID
		wc = append(wc, "id = :id")
	}

	if filter.Code != nil {
		data["code"] = filter.Code
		wc = append(wc, "code = :code")
	}

	if filter.ProvinceCode != nil {
		data["province_code"] = filter.ProvinceCode
		wc = append(wc, "province_code = :province_code")
	}

	if filter.IsActive != nil {
		data["is_active"] = filter.IsActive
		wc = append(wc, "is_active = :is_active")
	}

	// Default where clause to exclude deleted records
	wc = append(wc, "deleted_at IS NULL")

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
