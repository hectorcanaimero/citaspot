package worker

import (
	"fmt"
	"testing"
	"time"
)

func TestPartitionNameFormat(t *testing.T) {
	now := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	for i := 0; i <= 2; i++ {
		month := now.AddDate(0, i, 0)
		name := fmt.Sprintf("events_y%dm%02d", month.Year(), month.Month())
		expected := []string{"events_y2026m05", "events_y2026m06", "events_y2026m07"}
		if name != expected[i] {
			t.Fatalf("expected %s, got %s", expected[i], name)
		}
	}
}

func TestRetentionByPlan(t *testing.T) {
	cases := []struct {
		plan string
		days int
		ok   bool
	}{
		{"basic", 90, true},
		{"starter", 90, true},
		{"pro", 365, true},
		{"enterprise", 0, false},
	}
	for _, tc := range cases {
		days, ok := retentionByPlan[tc.plan]
		if ok != tc.ok {
			t.Fatalf("plan %s: expected ok=%v, got %v", tc.plan, tc.ok, ok)
		}
		if ok && days != tc.days {
			t.Fatalf("plan %s: expected %d days, got %d", tc.plan, tc.days, days)
		}
	}
}
