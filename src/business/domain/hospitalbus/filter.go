package hospitalbus

type QueryFilter struct {
	ID           *int
	Code         *string
	ProvinceCode *string
	IsActive     *bool
	OrderBy      *string
	Page         *int
	Limit        *int
}
