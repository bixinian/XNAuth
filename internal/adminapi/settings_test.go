package adminapi

import "testing"

func TestSanitizeFooterLinksAllowsEmptyConfiguration(t *testing.T) {
	links := sanitizeFooterLinks(nil)
	if len(links) != 0 {
		t.Fatalf("expected empty footer links, got %#v", links)
	}
}

func TestSanitizeFooterLinksDropsInvalidRowsWithoutDefaulting(t *testing.T) {
	links := sanitizeFooterLinks([]systemFooterLink{
		{Label: "", URL: "/login"},
		{Label: "无效地址", URL: "ftp://example.com"},
	})
	if len(links) != 0 {
		t.Fatalf("expected invalid footer links to be dropped, got %#v", links)
	}
}
