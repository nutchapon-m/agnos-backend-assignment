package staffdb

import (
	"bytes"
	"strings"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
)

func (s *Store) applyFilters(filter staffbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	var wc []string
	// Add more filters as needed
	if filter.ID != nil {
		data["id"] = filter.ID
		wc = append(wc, "id = :id")
	}

	if filter.UserID != nil {
		data["user_id"] = filter.UserID
		wc = append(wc, "user_id = :user_id")
	}

	// Default where clause to exclude deleted records
	wc = append(wc, "deleted_at IS NULL")

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
