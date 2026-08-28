package staffbus

type QueryFilter struct {
	ID      *int
	UserID  *int
	OrderBy *string
	Page    *int
	Limit   *int
}
