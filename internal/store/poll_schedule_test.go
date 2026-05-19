package store

import (
	"testing"
	"time"
)

func mustParseRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return x.UTC()
}

func ptrTime(ti time.Time) *time.Time {
	return &ti
}

func TestSubscriptionPollDue_Interval(t *testing.T) {
	t.Parallel()
	last := mustParseRFC3339(t, "2024-06-01T10:00:00Z")

	sub := &Subscription{
		PollIntervalMinutes: 30,
		LastFetchedAt:       ptrTime(last),
	}
	if SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T10:29:59Z")) {
		t.Fatal("expected not due")
	}
	if !SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T10:30:00Z")) {
		t.Fatal("expected due on boundary")
	}
}

func TestSubscriptionPollDue_IntervalNeverFetched(t *testing.T) {
	t.Parallel()
	created := mustParseRFC3339(t, "2024-06-01T10:00:00Z")
	sub := &Subscription{PollIntervalMinutes: 120, CreatedAt: created}
	if SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T11:59:59Z")) {
		t.Fatal("expected not due before first interval after creation")
	}
	if !SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T12:00:00Z")) {
		t.Fatal("expected due at created_at + interval")
	}
}

func TestSubscriptionPollDue_CronNeverFetched(t *testing.T) {
	t.Parallel()
	// 创建于 10:05，每小时整点触发 → 首次应在 11:00
	sub := &Subscription{
		PollCron:         "0 * * * *",
		PollCronTimezone: "UTC",
		CreatedAt:        mustParseRFC3339(t, "2024-06-01T10:05:00Z"),
	}
	if SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T10:30:00Z")) {
		t.Fatal("expected not due before first cron tick after creation")
	}
	if !SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T11:00:00Z")) {
		t.Fatal("expected due on first hourly tick after creation")
	}
}

func TestSubscriptionPollDue_IntervalNeverFetchedZeroInterval(t *testing.T) {
	t.Parallel()
	created := mustParseRFC3339(t, "2024-06-01T10:00:00Z")
	sub := &Subscription{PollIntervalMinutes: 0, CreatedAt: created}
	if SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T12:00:00Z")) {
		t.Fatal("expected not due when poll_interval_minutes is zero")
	}
}

func TestSubscriptionPollDue_NeverFetchedZeroCreatedAt(t *testing.T) {
	t.Parallel()
	sub := &Subscription{PollIntervalMinutes: 30}
	if SubscriptionPollDue(sub, time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("expected not due when created_at is missing")
	}
}

func TestSubscriptionPollDue_CrontabHourly(t *testing.T) {
	t.Parallel()
	sub := &Subscription{
		PollCron:      "0 * * * *",
		LastFetchedAt: ptrTime(mustParseRFC3339(t, "2024-06-01T10:05:00Z")),
	}
	if SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T10:30:00Z")) {
		t.Fatal("expected not due before next hour tick")
	}
	if !SubscriptionPollDue(sub, mustParseRFC3339(t, "2024-06-01T11:05:00Z")) {
		t.Fatal("expected due after hourly tick passes")
	}
}

func TestValidatePollSchedule_CronVersusInterval(t *testing.T) {
	t.Parallel()
	if err := validatePollSchedule(Subscription{
		Name:        "n",
		FeedURL:     "http://x",
		DownloadDir: "/tmp",
		PollCron:    "@hourly",
	}); err != nil {
		t.Fatalf("cron-only: unexpected err: %v", err)
	}
	if err := validatePollSchedule(Subscription{
		Name:                "n",
		FeedURL:             "http://x",
		DownloadDir:         "/tmp",
		PollIntervalMinutes: 0,
		PollCron:            "",
	}); err == nil {
		t.Fatal("expected interval validation error")
	}
	if err := validatePollSchedule(Subscription{
		Name:        "n",
		FeedURL:     "http://x",
		DownloadDir: "/tmp",
		PollCron:    "not a cron expression",
	}); err == nil {
		t.Fatal("expected cron parse error")
	}
}

func TestSubscriptionPollDue_CrontabTimezoneAsiaTokyo(t *testing.T) {
	t.Parallel()
	// Daily 23:59 JST; last fetched 2025-06-02 14:30 UTC (= 23:30 JST that day).
	sub := &Subscription{
		PollCron:         "59 23 * * *",
		PollCronTimezone: "Asia/Tokyo",
		LastFetchedAt:    ptrTime(mustParseRFC3339(t, "2025-06-02T14:30:00Z")),
	}
	if SubscriptionPollDue(sub, mustParseRFC3339(t, "2025-06-02T14:58:00Z")) {
		t.Fatal("expected not due before JST 23:59 tick")
	}
	if !SubscriptionPollDue(sub, mustParseRFC3339(t, "2025-06-02T15:01:00Z")) {
		t.Fatal("expected due after JST 23:59 on that calendar date")
	}
}

func TestValidatePollSchedule_DisallowEmbeddedTZDirective(t *testing.T) {
	t.Parallel()
	err := validatePollSchedule(Subscription{
		Name:        "n",
		FeedURL:     "http://x",
		DownloadDir: "/tmp",
		PollCron:    "TZ=UTC 0 * * * *",
	})
	if err == nil {
		t.Fatal("expected error for embedded TZ directive")
	}
}

func TestSubscriptionNextPollAt_Interval(t *testing.T) {
	t.Parallel()
	last := mustParseRFC3339(t, "2024-06-01T10:00:00Z")
	sub := &Subscription{
		PollIntervalMinutes: 30,
		LastFetchedAt:       ptrTime(last),
	}
	next, ok := SubscriptionNextPollAt(sub)
	if !ok {
		t.Fatal("expected ok")
	}
	want := mustParseRFC3339(t, "2024-06-01T10:30:00Z")
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestSubscriptionNextPollAt_CronNeverFetched(t *testing.T) {
	t.Parallel()
	sub := &Subscription{
		PollCron:         "0 * * * *",
		PollCronTimezone: "UTC",
		CreatedAt:        mustParseRFC3339(t, "2024-06-01T10:05:00Z"),
	}
	next, ok := SubscriptionNextPollAt(sub)
	if !ok {
		t.Fatal("expected ok")
	}
	want := mustParseRFC3339(t, "2024-06-01T11:00:00Z")
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestPreviewSubscriptionNextPoll_InvalidCron(t *testing.T) {
	t.Parallel()
	_, err := PreviewSubscriptionNextPoll(Subscription{
		Name:        "n",
		FeedURL:     "http://x",
		DownloadDir: "/tmp",
		PollCron:    "not valid",
	})
	if err == nil {
		t.Fatal("expected error for invalid cron")
	}
}

func TestPreviewSubscriptionNextPoll_Interval(t *testing.T) {
	t.Parallel()
	last := mustParseRFC3339(t, "2024-06-01T10:00:00Z")
	next, err := PreviewSubscriptionNextPoll(Subscription{
		PollIntervalMinutes: 30,
		LastFetchedAt:       ptrTime(last),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := mustParseRFC3339(t, "2024-06-01T10:30:00Z")
	if !next.Equal(want) {
		t.Fatalf("got %s want %s", next, want)
	}
}

func TestSubscriptionNextPollAt_MissingCreatedAt(t *testing.T) {
	t.Parallel()
	sub := &Subscription{PollIntervalMinutes: 30}
	if _, ok := SubscriptionNextPollAt(sub); ok {
		t.Fatal("expected not ok without created_at")
	}
}

func TestValidatePollSchedule_BadTimezone(t *testing.T) {
	t.Parallel()
	err := validatePollSchedule(Subscription{
		Name:             "n",
		FeedURL:          "http://x",
		DownloadDir:      "/tmp",
		PollCron:         "0 * * * *",
		PollCronTimezone: "Moon/Crater",
	})
	if err == nil {
		t.Fatal("expected invalid timezone error")
	}
}
