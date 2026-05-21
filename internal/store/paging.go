package store

const (
	DefaultPageSize = 30
	MaxPageSize     = 100
)

// NormalizePage 规范化页码与每页条数，并返回 SQL OFFSET。
func NormalizePage(page, pageSize int) (normPage, normPageSize, offset int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}
