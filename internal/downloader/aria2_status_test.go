package downloader

import "testing"

func TestParseAria2TaskStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		status  map[string]any
		want    Aria2TaskState
		wantErr string
	}{
		{name: "complete", status: map[string]any{"status": "complete"}, want: Aria2TaskComplete},
		{name: "active", status: map[string]any{"status": "active"}, want: Aria2TaskActive},
		{name: "waiting", status: map[string]any{"status": "waiting"}, want: Aria2TaskActive},
		{name: "error with message", status: map[string]any{"status": "error", "errorMessage": "disk full"}, want: Aria2TaskError, wantErr: "disk full"},
		{name: "error default", status: map[string]any{"status": "error"}, want: Aria2TaskError, wantErr: "aria2 下载失败"},
		{name: "removed", status: map[string]any{"status": "removed"}, want: Aria2TaskRemoved},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, errMsg := ParseAria2TaskStatus(tc.status)
			if got != tc.want {
				t.Fatalf("state = %v, want %v", got, tc.want)
			}
			if errMsg != tc.wantErr {
				t.Fatalf("errMsg = %q, want %q", errMsg, tc.wantErr)
			}
		})
	}
}
