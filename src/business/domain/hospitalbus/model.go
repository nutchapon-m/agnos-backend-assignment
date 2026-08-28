package hospitalbus

import "time"

type Hospital struct {
	ID           int
	Code         string
	Name         string
	ProvinceCode string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type NewHospital struct {
	Code         string
	Name         string
	ProvinceCode string
}
