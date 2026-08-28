package userbus

type QueryFilter struct {
	ID       *int
	Username *string
	OrderBy  *string
	Page     *int
	Limit    *int
}
