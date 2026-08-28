package hospitalpatientdb

import (
	"bytes"
	"strings"

	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
)

func (s *Store) applyFilters(filter hospitalpatientbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
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

	if filter.PatientID != nil {
		data["patient_id"] = filter.PatientID
		wc = append(wc, "patient_id = :patient_id")
	}

	if filter.HN != nil {
		data["hn"] = filter.HN
		wc = append(wc, "hn = :hn")
	}

	if filter.Status != nil {
		data["status"] = filter.Status
		wc = append(wc, "status = :status")
	}

	// Default where clause to exclude deleted records
	wc = append(wc, "deleted_at IS NULL")

	if len(wc) > 0 {
		buf.WriteString(" WHERE " + strings.Join(wc, " AND "))
	}
}
