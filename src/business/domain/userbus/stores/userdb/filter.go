package userdb

import (
	"bytes"
	"strings"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
)

func (s *Store) applyFilters(filter userbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string
	// Add more filters as needed
	if filter.ID != nil {
		data["id"] = filter.ID
		wc = append(wc, "id = :id")
	}

	// Default where clause to exclude deleted records
	wc = append(wc, "deleted_at IS NULL")

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
