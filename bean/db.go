package bean

type Page struct {
	Page  int64 `json:"page"`
	Size  int64 `json:"size"`
	Total int64 `json:"total"`
	Pages int64 `json:"pages"`
	Data  any   `json:"data"`
}
type PageGen struct {
	SQL       string
	Args      []any
	CountCols string
	FindCols  string
}
