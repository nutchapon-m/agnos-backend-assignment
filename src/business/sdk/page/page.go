package page

import (
	"fmt"
)

// Page represents the requested page and rows per page.
type Page struct {
	number int
	rows   int
}

// Parse parses the strings and validates the values are in reason.
func Parse(page int, rowsPerPage int) (Page, error) {
	number := 1
	if page != 0 {
		number = page
	}

	rows := 10
	if rowsPerPage != 0 {
		rows = rowsPerPage
	}

	if number <= 0 {
		return Page{}, fmt.Errorf("page value too small, must be larger than 0")
	}

	if rows <= 0 {
		return Page{}, fmt.Errorf("rows value too small, must be larger than 0")
	}

	if rows > 100 {
		return Page{}, fmt.Errorf("rows value too large, must be less than 100")
	}

	p := Page{
		number: number,
		rows:   rows,
	}

	return p, nil
}

// MustParse creates a paging value for testing.
func MustParse(page int, rowsPerPage int) Page {
	pg, err := Parse(page, rowsPerPage)
	if err != nil {
		panic(err)
	}

	return pg
}

// String implements the stringer interface.
func (p Page) String() string {
	return fmt.Sprintf("page: %d rows: %d", p.number, p.rows)
}

// Number returns the page number.
func (p Page) Number() int {
	return p.number
}

// RowsPerPage returns the rows per page.
func (p Page) RowsPerPage() int {
	return p.rows
}

func (p *Page) IsZero() bool {
	return p.rows == 0 && p.number == 0
}
