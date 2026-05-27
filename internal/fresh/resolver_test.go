package fresh

import (
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	const ageDays = 7
	now := time.Now()

	tests := []struct {
		name      string
		published time.Time
		want      Risk
	}{
		{"1 day old is HIGH", now.Add(-1 * 24 * time.Hour), RiskHigh},
		{"3 days old is HIGH", now.Add(-3 * 24 * time.Hour), RiskHigh},
		{"4 days old is WARN", now.Add(-4 * 24 * time.Hour), RiskWarn},
		{"6 days old is WARN", now.Add(-6 * 24 * time.Hour), RiskWarn},
		{"8 days old is OK", now.Add(-8 * 24 * time.Hour), RiskOK},
		{"30 days old is OK", now.Add(-30 * 24 * time.Hour), RiskOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classify(tc.published, ageDays)
			if got != tc.want {
				t.Errorf("classify(published %v ago, ageDays=%d) = %v, want %v",
					time.Since(tc.published).Round(time.Hour), ageDays, got, tc.want)
			}
		})
	}
}

func TestClassifyCustomGate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		ageDays   int
		published time.Time
		want      Risk
	}{
		{14, now.Add(-3 * 24 * time.Hour), RiskHigh}, // 3d < 7d (14/2)
		{14, now.Add(-9 * 24 * time.Hour), RiskWarn}, // 7d ≤ 9d < 14d
		{14, now.Add(-20 * 24 * time.Hour), RiskOK},  // 20d ≥ 14d
		{1, now.Add(-30 * time.Minute), RiskHigh},    // <12h with a 1-day gate
	}

	for _, tc := range tests {
		got := classify(tc.published, tc.ageDays)
		if got != tc.want {
			t.Errorf("classify(ageDays=%d) = %v, want %v", tc.ageDays, got, tc.want)
		}
	}
}
