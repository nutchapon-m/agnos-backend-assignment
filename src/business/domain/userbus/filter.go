package userbus

type QueryFilter struct {
	ID      *int
	OrderBy *string
	Page    *int
	Limit   *int
}
