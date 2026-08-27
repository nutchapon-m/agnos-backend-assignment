package hospitaldb

type hospital struct {
	ID    string `db:"id"`
	Title string `db:"title"`
	Size  string `db:"size"`
}
