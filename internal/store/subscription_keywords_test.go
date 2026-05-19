package store

import "testing"

func TestValidateSubscription_InvalidIncludeRegex(t *testing.T) {
	t.Parallel()
	err := validateSubscription(Subscription{
		Name:                "n",
		FeedURL:             "http://x",
		Enabled:             true,
		PollIntervalMinutes: 10,
		DownloadDir:         "/tmp",
		IncludeKeywords:     "(",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
