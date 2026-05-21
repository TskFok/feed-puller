package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestParsePageParams(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/api/subscriptions?page=2&page_size=50", nil)
	p := parsePageParams(req)
	if p.Page != 2 || p.PageSize != 50 {
		t.Fatalf("got page=%d page_size=%d", p.Page, p.PageSize)
	}

	req = httptest.NewRequest("GET", "/api/subscriptions", nil)
	p = parsePageParams(req)
	if p.Page != 1 || p.PageSize != defaultPageSize {
		t.Fatalf("defaults: got page=%d page_size=%d", p.Page, p.PageSize)
	}

	req = httptest.NewRequest("GET", "/api/subscriptions?page=0&page_size=999", nil)
	p = parsePageParams(req)
	if p.Page != 1 || p.PageSize != maxPageSize {
		t.Fatalf("clamped: got page=%d page_size=%d", p.Page, p.PageSize)
	}
}
