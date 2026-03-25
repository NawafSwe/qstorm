package metric

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCollector_Record(t *testing.T) {
	tests := map[string]struct {
		records []struct {
			duration time.Duration
			err      error
		}
		wantSnapshot Snapshot
	}{
		"single success": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{duration: 5 * time.Millisecond},
			},
			wantSnapshot: Snapshot{SuccessCount: 1, ErrorCount: 0},
		},

		"single error": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{err: errors.New("fail")},
			},
			wantSnapshot: Snapshot{SuccessCount: 0, ErrorCount: 1},
		},

		"mixed successes and errors": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{duration: 2 * time.Millisecond},
				{duration: 4 * time.Millisecond},
				{err: errors.New("fail")},
				{duration: 6 * time.Millisecond},
				{err: errors.New("fail again")},
			},
			wantSnapshot: Snapshot{SuccessCount: 3, ErrorCount: 2},
		},

		"no records": {
			records:      nil,
			wantSnapshot: Snapshot{SuccessCount: 0, ErrorCount: 0},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := NewCollector()
			for _, r := range tc.records {
				c.Record(r.duration, r.err)
			}
			assert.Equal(t, tc.wantSnapshot, c.Snapshot())
		})
	}
}

func TestCollector_Summary(t *testing.T) {
	tests := map[string]struct {
		records []struct {
			duration time.Duration
			err      error
		}
		assertFunc func(t *testing.T, s Summary)
	}{
		"no records returns zero summary": {
			records: nil,
			assertFunc: func(t *testing.T, s Summary) {
				assert.Equal(t, int64(0), s.SuccessCount)
				assert.Equal(t, int64(0), s.ErrorCount)
				assert.Equal(t, float64(0), s.SuccessRate)
				assert.Equal(t, float64(0), s.FailureRate)
				assert.Empty(t, s.ErrorsOverview)
			},
		},

		"all successes": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{duration: 10 * time.Millisecond},
				{duration: 20 * time.Millisecond},
				{duration: 30 * time.Millisecond},
			},
			assertFunc: func(t *testing.T, s Summary) {
				assert.Equal(t, int64(3), s.SuccessCount)
				assert.Equal(t, int64(0), s.ErrorCount)
				assert.Equal(t, float64(100), s.SuccessRate)
				assert.Equal(t, float64(0), s.FailureRate)
				assert.Greater(t, s.AverageLatency, float64(0))
				assert.Greater(t, s.P50Latency, float64(0))
				assert.Greater(t, s.P90Latency, float64(0))
				assert.Greater(t, s.P99Latency, float64(0))
				assert.Empty(t, s.ErrorsOverview)
			},
		},

		"all errors": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{err: errors.New("e1")},
				{err: errors.New("e2")},
			},
			assertFunc: func(t *testing.T, s Summary) {
				assert.Equal(t, int64(0), s.SuccessCount)
				assert.Equal(t, int64(2), s.ErrorCount)
				assert.Equal(t, float64(0), s.SuccessRate)
				assert.Equal(t, float64(100), s.FailureRate)
				assert.Equal(t, map[string]int{"e1": 1, "e2": 1}, s.ErrorsOverview)
			},
		},

		"errors overview aggregates by message": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{err: errors.New("connection refused")},
				{err: errors.New("connection refused")},
				{err: errors.New("context deadline exceeded")},
				{err: errors.New("connection refused")},
			},
			assertFunc: func(t *testing.T, s Summary) {
				assert.Equal(t, int64(4), s.ErrorCount)
				assert.Equal(t, map[string]int{
					"connection refused":        3,
					"context deadline exceeded": 1,
				}, s.ErrorsOverview)
			},
		},

		"rates calculated correctly": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{duration: 5 * time.Millisecond},
				{duration: 5 * time.Millisecond},
				{duration: 5 * time.Millisecond},
				{err: errors.New("fail")},
			},
			assertFunc: func(t *testing.T, s Summary) {
				assert.Equal(t, int64(3), s.SuccessCount)
				assert.Equal(t, int64(1), s.ErrorCount)
				assert.Equal(t, float64(75), s.SuccessRate)
				assert.Equal(t, float64(25), s.FailureRate)
				assert.Equal(t, map[string]int{"fail": 1}, s.ErrorsOverview)
			},
		},

		"latency percentiles ordered": {
			records: []struct {
				duration time.Duration
				err      error
			}{
				{duration: 1 * time.Millisecond},
				{duration: 5 * time.Millisecond},
				{duration: 10 * time.Millisecond},
				{duration: 50 * time.Millisecond},
				{duration: 100 * time.Millisecond},
				{duration: 200 * time.Millisecond},
				{duration: 500 * time.Millisecond},
				{duration: 1 * time.Second},
			},
			assertFunc: func(t *testing.T, s Summary) {
				assert.LessOrEqual(t, s.P50Latency, s.P75Latency)
				assert.LessOrEqual(t, s.P75Latency, s.P90Latency)
				assert.LessOrEqual(t, s.P90Latency, s.P99Latency)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := NewCollector()
			for _, r := range tc.records {
				c.Record(r.duration, r.err)
			}
			tc.assertFunc(t, c.Summary())
		})
	}
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	c := NewCollector()
	done := make(chan struct{})

	for range 100 {
		go func() {
			c.Record(5*time.Millisecond, nil)
			c.Snapshot()
			done <- struct{}{}
		}()
	}
	for range 100 {
		<-done
	}

	s := c.Summary()
	assert.Equal(t, int64(100), s.SuccessCount)
}
