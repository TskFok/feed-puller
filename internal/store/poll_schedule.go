package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// NormalizePollCron 返回首尾去空后的 Cron 表达式；空字符串表示不按 crontab 调度。
func NormalizePollCron(expr string) string {
	return strings.TrimSpace(expr)
}

// NormalizePollCronTimezone 返回用于解释 crontab 的 IANA 时区名；空串视为 UTC（与存量数据兼容）。
func NormalizePollCronTimezone(name string) string {
	s := strings.TrimSpace(name)
	if s == "" {
		return "UTC"
	}
	return s
}

func hasCronTZDirective(expr string) bool {
	up := strings.ToUpper(strings.TrimSpace(expr))
	return strings.HasPrefix(up, "TZ=") || strings.HasPrefix(up, "CRON_TZ=")
}

func parseCronSchedule(expr string, tzIANA string) (cron.Schedule, error) {
	expr = NormalizePollCron(expr)
	if expr == "" {
		return nil, fmt.Errorf("empty cron expression")
	}
	tz := NormalizePollCronTimezone(tzIANA)
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}
	if hasCronTZDirective(expr) {
		return nil, fmt.Errorf("expression must not contain TZ=/CRON_TZ= directives")
	}
	return cron.ParseStandard("CRON_TZ=" + tz + " " + expr)
}

// SubscriptionPollDue 判断在给定时间点是否应触发拉取。
// 「当前时刻」与其它时间戳比较使用 UTC；
// Crontab 在 poll_cron_timezone（IANA）下解释字段含义；表达式内勿再写 TZ=/CRON_TZ=。
// 从未拉取过（last_fetched_at 为空）时，从 created_at 起算下一次调度，避免新建订阅立刻被拉取。
func SubscriptionPollDue(sub *Subscription, now time.Time) bool {
	expr := NormalizePollCron(sub.PollCron)
	nowUTC := now.UTC()

	if sub.LastFetchedAt == nil {
		if sub.CreatedAt.IsZero() {
			return false
		}
		created := sub.CreatedAt.UTC()
		if expr != "" {
			sched, err := parseCronSchedule(expr, sub.PollCronTimezone)
			if err != nil {
				return false
			}
			firstFire := sched.Next(created)
			return !firstFire.After(nowUTC)
		}
		deadline := created.Add(time.Duration(sub.PollIntervalMinutes) * time.Minute)
		return !deadline.After(nowUTC)
	}

	if expr != "" {
		sched, err := parseCronSchedule(expr, sub.PollCronTimezone)
		if err != nil {
			return false
		}
		next := sched.Next(sub.LastFetchedAt.UTC())
		return !next.After(nowUTC)
	}
	deadline := sub.LastFetchedAt.UTC().Add(time.Duration(sub.PollIntervalMinutes) * time.Minute)
	return !deadline.After(nowUTC)
}

// SubscriptionNextPollAt 返回下一次计划拉取的 UTC 时间（与 SubscriptionPollDue 使用相同锚点）。
// ok 为 false 表示无法推算（如 created_at 缺失、cron 无效或间隔 ≤ 0）。
func SubscriptionNextPollAt(sub *Subscription) (time.Time, bool) {
	expr := NormalizePollCron(sub.PollCron)

	var anchor time.Time
	if sub.LastFetchedAt == nil {
		if sub.CreatedAt.IsZero() {
			return time.Time{}, false
		}
		anchor = sub.CreatedAt.UTC()
	} else {
		anchor = sub.LastFetchedAt.UTC()
	}

	if expr != "" {
		sched, err := parseCronSchedule(expr, sub.PollCronTimezone)
		if err != nil {
			return time.Time{}, false
		}
		return sched.Next(anchor), true
	}
	if sub.PollIntervalMinutes <= 0 {
		return time.Time{}, false
	}
	return anchor.Add(time.Duration(sub.PollIntervalMinutes) * time.Minute), true
}

// ApplySubscriptionNextPoll 填充 sub.NextPollAt（仅用于 API 响应，不入库）。
func ApplySubscriptionNextPoll(sub *Subscription, _ time.Time) {
	t, err := PreviewSubscriptionNextPoll(*sub)
	if err != nil {
		sub.NextPollAt = nil
		return
	}
	sub.NextPollAt = t
}

// PreviewSubscriptionNextPoll 根据调度配置推算下次拉取时间（不读写数据库）。
func PreviewSubscriptionNextPoll(sub Subscription) (*time.Time, error) {
	if err := validatePollSchedule(sub); err != nil {
		return nil, err
	}
	t, ok := SubscriptionNextPollAt(&sub)
	if !ok {
		return nil, fmt.Errorf("无法推算下次拉取时间")
	}
	return &t, nil
}

func validatePollSchedule(sub Subscription) error {
	expr := NormalizePollCron(sub.PollCron)
	if expr != "" {
		if hasCronTZDirective(expr) {
			return fmt.Errorf("请勿在表达式内写 TZ=/CRON_TZ=，请使用时区字段（IANA 名称）")
		}
		_, err := parseCronSchedule(expr, sub.PollCronTimezone)
		if err != nil {
			return fmt.Errorf("无效的 crontab 或时区: %w", err)
		}
		return nil
	}
	if sub.PollIntervalMinutes <= 0 {
		return fmt.Errorf("拉取间隔必须大于 0")
	}
	return nil
}
