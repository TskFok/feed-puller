package paths

import "testing"

func TestMapper_Map(t *testing.T) {
	t.Parallel()
	m := NewMapper("/Users/demo/Downloads", "/downloads")
	cases := []struct {
		in   string
		want string
	}{
		{"/Users/demo/Downloads/anime/a.mp4", "/downloads/anime/a.mp4"},
		{"/Users/demo/Downloads", "/downloads"},
		{"/other/a.mp4", "/other/a.mp4"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := m.Map(tc.in); got != tc.want {
			t.Fatalf("Map(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMapper_Disabled(t *testing.T) {
	t.Parallel()
	m := NewMapper("", "/downloads")
	if got := m.Map("/Users/a.mp4"); got != "/Users/a.mp4" {
		t.Fatalf("got %q", got)
	}
}
