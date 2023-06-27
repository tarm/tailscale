// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package tstest

import (
	"testing"
	"time"

	"golang.org/x/exp/slices"
)

func TestClockWithDefinedStartTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start time.Time
		step  time.Duration
		wants []time.Time // The return values of sequential calls to Now().
	}{
		{
			name:  "increment ms",
			start: time.Unix(12345, 1000),
			step:  1000,
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 2000),
				time.Unix(12345, 3000),
				time.Unix(12345, 4000),
			},
		},
		{
			name:  "increment second",
			start: time.Unix(12345, 1000),
			step:  time.Second,
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12346, 1000),
				time.Unix(12347, 1000),
				time.Unix(12348, 1000),
			},
		},
		{
			name:  "no increment",
			start: time.Unix(12345, 1000),
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 1000),
				time.Unix(12345, 1000),
				time.Unix(12345, 1000),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clock := NewClockBuilder().Start(tt.start).Step(tt.step).Finish()

			if start := clock.Start(); !start.Equal(tt.start) {
				t.Errorf("clock has start %v, want %v", start, tt.start)
			}
			if step := clock.Step(); step != tt.step {
				t.Errorf("clock has step %v, want %v", step, tt.step)
			}

			for i := range tt.wants {
				if got := clock.Now(); !got.Equal(tt.wants[i]) {
					t.Errorf("step %v: clock.Now() = %v, want %v", i, got, tt.wants[i])
				}
				if got := clock.ConstantNow(); !got.Equal(tt.wants[i]) {
					t.Errorf("step %v: clock.ConstantNow() = %v, want %v", i, got, tt.wants[i])
				}
			}
		})
	}
}

func TestClockWithDefaultStartTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		step  time.Duration
		wants []time.Duration // The return values of sequential calls to Now() after added to Start()
	}{
		{
			name: "increment ms",
			step: 1000,
			wants: []time.Duration{
				0,
				1000,
				2000,
				3000,
			},
		},
		{
			name: "increment second",
			step: time.Second,
			wants: []time.Duration{
				0 * time.Second,
				1 * time.Second,
				2 * time.Second,
				3 * time.Second,
			},
		},
		{
			name:  "no increment",
			wants: []time.Duration{0, 0, 0, 0},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clock := NewClockBuilder().Step(tt.step).Finish()
			start := clock.Start()

			if step := clock.Step(); step != tt.step {
				t.Errorf("clock has step %v, want %v", step, tt.step)
			}

			for i := range tt.wants {
				want := start.Add(tt.wants[i])
				if got := clock.Now(); !got.Equal(want) {
					t.Errorf("step %v: clock.Now() = %v, want %v", i, got, tt.wants[i])
				}
				if got := clock.ConstantNow(); !got.Equal(want) {
					t.Errorf("step %v: clock.ConstantNow() = %v, want %v", i, got, tt.wants[i])
				}
			}
		})
	}
}

func TestNewClock(t *testing.T) {
	t.Parallel()

	clock := NewClock()
	start := clock.Start()

	if step := clock.Step(); step != 0 {
		t.Errorf("clock has step %v, want 0", step)
	}

	for i := 0; i < 10; i++ {
		if got := clock.Now(); !got.Equal(start) {
			t.Errorf("step %v: clock.Now() = %v, want %v", i, got, start)
		}
		if got := clock.ConstantNow(); !got.Equal(start) {
			t.Errorf("step %v: clock.ConstantNow() = %v, want %v", i, got, start)
		}
	}
}

func TestClockSetStep(t *testing.T) {
	t.Parallel()

	type stepInfo struct {
		when int
		step time.Duration
	}

	tests := []struct {
		name        string
		start       time.Time
		step        time.Duration
		stepChanges []stepInfo
		wants       []time.Time // The return values of sequential calls to Now().
	}{
		{
			name:  "increment ms then s",
			start: time.Unix(12345, 1000),
			step:  1000,
			stepChanges: []stepInfo{
				{
					when: 4,
					step: time.Second,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 2000),
				time.Unix(12345, 3000),
				time.Unix(12345, 4000),
				time.Unix(12346, 4000),
				time.Unix(12347, 4000),
				time.Unix(12348, 4000),
				time.Unix(12349, 4000),
			},
		},
		{
			name:  "multiple changes over time",
			start: time.Unix(12345, 1000),
			step:  1,
			stepChanges: []stepInfo{
				{
					when: 2,
					step: time.Second,
				},
				{
					when: 4,
					step: 0,
				},
				{
					when: 6,
					step: 1000,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 1001),
				time.Unix(12346, 1001),
				time.Unix(12347, 1001),
				time.Unix(12347, 1001),
				time.Unix(12347, 1001),
				time.Unix(12347, 2001),
				time.Unix(12347, 3001),
			},
		},
		{
			name:  "multiple changes at once",
			start: time.Unix(12345, 1000),
			step:  1,
			stepChanges: []stepInfo{
				{
					when: 2,
					step: time.Second,
				},
				{
					when: 2,
					step: 0,
				},
				{
					when: 2,
					step: 1000,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 1001),
				time.Unix(12345, 2001),
				time.Unix(12345, 3001),
			},
		},
		{
			name:  "changes at start",
			start: time.Unix(12345, 1000),
			step:  0,
			stepChanges: []stepInfo{
				{
					when: 0,
					step: time.Second,
				},
				{
					when: 0,
					step: 1000,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 2000),
				time.Unix(12345, 3000),
				time.Unix(12345, 4000),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clock := NewClockBuilder().Start(tt.start).Step(tt.step).Finish()
			wantStep := tt.step
			changeIndex := 0

			for i := range tt.wants {
				for len(tt.stepChanges) > changeIndex && tt.stepChanges[changeIndex].when == i {
					wantStep = tt.stepChanges[changeIndex].step
					clock.SetStep(wantStep)
					changeIndex++
				}

				if start := clock.Start(); !start.Equal(tt.start) {
					t.Errorf("clock has start %v, want %v", start, tt.start)
				}
				if step := clock.Step(); step != wantStep {
					t.Errorf("clock has step %v, want %v", step, tt.step)
				}

				if got := clock.Now(); !got.Equal(tt.wants[i]) {
					t.Errorf("step %v: clock.Now() = %v, want %v", i, got, tt.wants[i])
				}
				if got := clock.ConstantNow(); !got.Equal(tt.wants[i]) {
					t.Errorf("step %v: clock.ConstantNow() = %v, want %v", i, got, tt.wants[i])
				}
			}
		})
	}
}

func TestClockAdvance(t *testing.T) {
	t.Parallel()

	type advanceInfo struct {
		when    int
		advance time.Duration
	}

	tests := []struct {
		name     string
		start    time.Time
		step     time.Duration
		advances []advanceInfo
		wants    []time.Time // The return values of sequential calls to Now().
	}{
		{
			name:  "increment ms then advance 1s",
			start: time.Unix(12345, 1000),
			step:  1000,
			advances: []advanceInfo{
				{
					when:    4,
					advance: time.Second,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 2000),
				time.Unix(12345, 3000),
				time.Unix(12345, 4000),
				time.Unix(12346, 4000),
				time.Unix(12346, 5000),
				time.Unix(12346, 6000),
				time.Unix(12346, 7000),
			},
		},
		{
			name:  "multiple advances over time",
			start: time.Unix(12345, 1000),
			step:  1,
			advances: []advanceInfo{
				{
					when:    2,
					advance: time.Second,
				},
				{
					when:    4,
					advance: 0,
				},
				{
					when:    6,
					advance: 1000,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 1001),
				time.Unix(12346, 1001),
				time.Unix(12346, 1002),
				time.Unix(12346, 1002),
				time.Unix(12346, 1003),
				time.Unix(12346, 2003),
				time.Unix(12346, 2004),
			},
		},
		{
			name:  "multiple advances at once",
			start: time.Unix(12345, 1000),
			step:  1,
			advances: []advanceInfo{
				{
					when:    2,
					advance: time.Second,
				},
				{
					when:    2,
					advance: 0,
				},
				{
					when:    2,
					advance: 1000,
				},
			},
			wants: []time.Time{
				time.Unix(12345, 1000),
				time.Unix(12345, 1001),
				time.Unix(12346, 2001),
				time.Unix(12346, 2002),
			},
		},
		{
			name:  "changes at start",
			start: time.Unix(12345, 1000),
			step:  5,
			advances: []advanceInfo{
				{
					when:    0,
					advance: time.Second,
				},
				{
					when:    0,
					advance: 1000,
				},
			},
			wants: []time.Time{
				time.Unix(12346, 2000),
				time.Unix(12346, 2005),
				time.Unix(12346, 2010),
				time.Unix(12346, 2015),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clock := NewClockBuilder().Start(tt.start).Step(tt.step).Finish()
			wantStep := tt.step
			changeIndex := 0

			for i := range tt.wants {
				for len(tt.advances) > changeIndex && tt.advances[changeIndex].when == i {
					clock.Advance(tt.advances[changeIndex].advance)
					changeIndex++
				}

				if start := clock.Start(); !start.Equal(tt.start) {
					t.Errorf("clock has start %v, want %v", start, tt.start)
				}
				if step := clock.Step(); step != wantStep {
					t.Errorf("clock has step %v, want %v", step, tt.step)
				}

				if got := clock.Now(); !got.Equal(tt.wants[i]) {
					t.Errorf("step %v: clock.Now() = %v, want %v", i, got, tt.wants[i])
				}
				if got := clock.ConstantNow(); !got.Equal(tt.wants[i]) {
					t.Errorf("step %v: clock.ConstantNow() = %v, want %v", i, got, tt.wants[i])
				}
			}
		})
	}
}

func expectNoTicks(t *testing.T, tickC <-chan time.Time) {
	t.Helper()
	select {
	case tick := <-tickC:
		t.Errorf("wanted no ticks, got %v", tick)
	default:
	}
}

func TestSingleTicker(t *testing.T) {
	t.Parallel()

	type testStep struct {
		stop          bool
		reset         time.Duration
		resetAbsolute time.Time
		setStep       time.Duration
		advance       time.Duration
		wantTime      time.Time
		wantTicks     []time.Time
	}

	tests := []struct {
		name        string
		start       time.Time
		step        time.Duration
		period      time.Duration
		channelSize int
		steps       []testStep
	}{
		{
			name:   "no tick advance",
			start:  time.Unix(12345, 0),
			period: time.Second,
			steps: []testStep{
				{
					advance:  time.Second - 1,
					wantTime: time.Unix(12345, 999_999_999),
				},
			},
		},
		{
			name:   "no tick step",
			start:  time.Unix(12345, 0),
			step:   time.Second - 1,
			period: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12345, 999_999_999),
				},
			},
		},
		{
			name:   "single tick advance exact",
			start:  time.Unix(12345, 0),
			period: time.Second,
			steps: []testStep{
				{
					advance:   time.Second,
					wantTime:  time.Unix(12346, 0),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
			},
		},
		{
			name:   "single tick advance extra",
			start:  time.Unix(12345, 0),
			period: time.Second,
			steps: []testStep{
				{
					advance:   time.Second + 1,
					wantTime:  time.Unix(12346, 1),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
			},
		},
		{
			name:   "single tick step exact",
			start:  time.Unix(12345, 0),
			step:   time.Second,
			period: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime:  time.Unix(12346, 0),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
			},
		},
		{
			name:   "single tick step extra",
			start:  time.Unix(12345, 0),
			step:   time.Second + 1,
			period: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime:  time.Unix(12346, 1),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
			},
		},
		{
			name:   "single tick per advance",
			start:  time.Unix(12345, 0),
			period: 3 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					advance:   4 * time.Second,
					wantTime:  time.Unix(12349, 0),
					wantTicks: []time.Time{time.Unix(12348, 0)},
				},
				{
					advance:   2 * time.Second,
					wantTime:  time.Unix(12351, 0),
					wantTicks: []time.Time{time.Unix(12351, 0)},
				},
				{
					advance:  2 * time.Second,
					wantTime: time.Unix(12353, 0),
				},
				{
					advance:   2 * time.Second,
					wantTime:  time.Unix(12355, 0),
					wantTicks: []time.Time{time.Unix(12354, 0)},
				},
			},
		},
		{
			name:   "single tick per step",
			start:  time.Unix(12345, 0),
			step:   2 * time.Second,
			period: 3 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
				},
				{
					wantTime:  time.Unix(12349, 0),
					wantTicks: []time.Time{time.Unix(12348, 0)},
				},
				{
					wantTime:  time.Unix(12351, 0),
					wantTicks: []time.Time{time.Unix(12351, 0)},
				},
				{
					wantTime: time.Unix(12353, 0),
				},
				{
					wantTime:  time.Unix(12355, 0),
					wantTicks: []time.Time{time.Unix(12354, 0)},
				},
			},
		},
		{
			name:        "multiple tick per advance",
			start:       time.Unix(12345, 0),
			period:      time.Second,
			channelSize: 3,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					advance:  2 * time.Second,
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12346, 0),
						time.Unix(12347, 0),
					},
				},
				{
					advance:  4 * time.Second,
					wantTime: time.Unix(12351, 0),
					wantTicks: []time.Time{
						time.Unix(12348, 0),
						time.Unix(12349, 0),
						time.Unix(12350, 0),
						// fourth tick dropped due to channel size
					},
				},
			},
		},
		{
			name:        "multiple tick per step",
			start:       time.Unix(12345, 0),
			step:        3 * time.Second,
			period:      2 * time.Second,
			channelSize: 3,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12348, 0),
					wantTicks: []time.Time{
						time.Unix(12347, 0),
					},
				},
				{
					wantTime: time.Unix(12351, 0),
					wantTicks: []time.Time{
						time.Unix(12349, 0),
						time.Unix(12351, 0),
					},
				},
				{
					wantTime: time.Unix(12354, 0),
					wantTicks: []time.Time{
						time.Unix(12353, 0),
					},
				},
				{
					wantTime: time.Unix(12357, 0),
					wantTicks: []time.Time{
						time.Unix(12355, 0),
						time.Unix(12357, 0),
					},
				},
			},
		},
		{
			name:        "stop",
			start:       time.Unix(12345, 0),
			step:        2 * time.Second,
			period:      time.Second,
			channelSize: 3,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12346, 0),
						time.Unix(12347, 0),
					},
				},
				{
					stop:     true,
					wantTime: time.Unix(12349, 0),
				},
				{
					wantTime: time.Unix(12351, 0),
				},
				{
					advance:  10 * time.Second,
					wantTime: time.Unix(12361, 0),
				},
			},
		},
		{
			name:   "reset while running",
			start:  time.Unix(12345, 0),
			period: 2 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					advance:  time.Second,
					wantTime: time.Unix(12346, 0),
				},
				{
					advance:  time.Second,
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12347, 0),
					},
				},
				{
					advance:  time.Second,
					reset:    time.Second,
					wantTime: time.Unix(12348, 0),
					wantTicks: []time.Time{
						time.Unix(12348, 0),
					},
				},
				{
					setStep:  5 * time.Second,
					reset:    10 * time.Second,
					wantTime: time.Unix(12353, 0),
				},
				{
					wantTime: time.Unix(12358, 0),
					wantTicks: []time.Time{
						time.Unix(12358, 0),
					},
				},
			},
		},
		{
			name:   "reset while stopped",
			start:  time.Unix(12345, 0),
			step:   time.Second,
			period: 2 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12346, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12347, 0),
					},
				},
				{
					stop:     true,
					wantTime: time.Unix(12348, 0),
				},
				{
					wantTime: time.Unix(12349, 0),
				},
				{
					reset:    time.Second,
					wantTime: time.Unix(12350, 0),
					wantTicks: []time.Time{
						time.Unix(12350, 0),
					},
				},
				{
					wantTime: time.Unix(12351, 0),
					wantTicks: []time.Time{
						time.Unix(12351, 0),
					},
				},
			},
		},
		{
			name:   "reset absolute",
			start:  time.Unix(12345, 0),
			step:   time.Second,
			period: 2 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12346, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12347, 0),
					},
				},
				{
					reset:         time.Second,
					resetAbsolute: time.Unix(12354, 50),
					advance:       7 * time.Second,
					wantTime:      time.Unix(12354, 0),
				},
				{
					wantTime: time.Unix(12355, 0),
					wantTicks: []time.Time{
						time.Unix(12354, 50),
					},
				},
				{
					wantTime: time.Unix(12356, 0),
					wantTicks: []time.Time{
						time.Unix(12355, 50),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clock := NewClockBuilder().
				Start(tt.start).
				Step(tt.step).
				TimerChannelSize(tt.channelSize).
				Finish()
			tc, tickC := clock.NewTicker(tt.period)
			tickControl := tc.(*Ticker)

			t.Cleanup(tickControl.Stop)

			expectNoTicks(t, tickC)

			for i, step := range tt.steps {
				if step.stop {
					tickControl.Stop()
				}

				if !step.resetAbsolute.IsZero() {
					tickControl.ResetAbsolute(step.resetAbsolute, step.reset)
				} else if step.reset > 0 {
					tickControl.Reset(step.reset)
				}

				if step.setStep > 0 {
					clock.SetStep(step.setStep)
				}

				if step.advance > 0 {
					clock.Advance(step.advance)
				}

				if now := clock.Now(); !step.wantTime.IsZero() && !now.Equal(step.wantTime) {
					t.Errorf("step %v now = %v, want %v", i, now, step.wantTime)
				}

				for j, want := range step.wantTicks {
					select {
					case tick := <-tickC:
						if tick.Equal(want) {
							continue
						}
						t.Errorf("step %v tick %v = %v, want %v", i, j, tick, want)
					default:
						t.Errorf("step %v tick %v missing", i, j)
					}
				}

				expectNoTicks(t, tickC)
			}
		})
	}
}

func TestSingleTimer(t *testing.T) {
	t.Parallel()

	type testStep struct {
		stop          bool
		stopReturn    bool // expected return value for Stop() if stop is true
		reset         time.Duration
		resetAbsolute time.Time
		resetReturn   bool // expected return value for Reset() or ResetAbsolute()
		setStep       time.Duration
		advance       time.Duration
		wantTime      time.Time
		wantTicks     []time.Time
	}

	tests := []struct {
		name  string
		start time.Time
		step  time.Duration
		delay time.Duration
		steps []testStep
	}{
		{
			name:  "no tick advance",
			start: time.Unix(12345, 0),
			delay: time.Second,
			steps: []testStep{
				{
					advance:  time.Second - 1,
					wantTime: time.Unix(12345, 999_999_999),
				},
			},
		},
		{
			name:  "no tick step",
			start: time.Unix(12345, 0),
			step:  time.Second - 1,
			delay: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12345, 999_999_999),
				},
			},
		},
		{
			name:  "single tick advance exact",
			start: time.Unix(12345, 0),
			delay: time.Second,
			steps: []testStep{
				{
					advance:   time.Second,
					wantTime:  time.Unix(12346, 0),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
				{
					advance:  time.Second,
					wantTime: time.Unix(12347, 0),
				},
			},
		},
		{
			name:  "single tick advance extra",
			start: time.Unix(12345, 0),
			delay: time.Second,
			steps: []testStep{
				{
					advance:   time.Second + 1,
					wantTime:  time.Unix(12346, 1),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
				{
					advance:  time.Second,
					wantTime: time.Unix(12347, 1),
				},
			},
		},
		{
			name:  "single tick step exact",
			start: time.Unix(12345, 0),
			step:  time.Second,
			delay: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime:  time.Unix(12346, 0),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
				{
					wantTime: time.Unix(12347, 0),
				},
			},
		},
		{
			name:  "single tick step extra",
			start: time.Unix(12345, 0),
			step:  time.Second + 1,
			delay: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime:  time.Unix(12346, 1),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
				{
					wantTime: time.Unix(12347, 2),
				},
			},
		},
		{
			name:  "reset for single tick per advance",
			start: time.Unix(12345, 0),
			delay: 3 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					advance:   4 * time.Second,
					wantTime:  time.Unix(12349, 0),
					wantTicks: []time.Time{time.Unix(12348, 0)},
				},
				{
					resetAbsolute: time.Unix(12351, 0),
					advance:       2 * time.Second,
					wantTime:      time.Unix(12351, 0),
					wantTicks:     []time.Time{time.Unix(12351, 0)},
				},
				{
					reset:    3 * time.Second,
					advance:  2 * time.Second,
					wantTime: time.Unix(12353, 0),
				},
				{
					advance:   2 * time.Second,
					wantTime:  time.Unix(12355, 0),
					wantTicks: []time.Time{time.Unix(12354, 0)},
				},
				{
					advance:  10 * time.Second,
					wantTime: time.Unix(12365, 0),
				},
			},
		},
		{
			name:  "reset for single tick per step",
			start: time.Unix(12345, 0),
			step:  2 * time.Second,
			delay: 3 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
				},
				{
					wantTime:  time.Unix(12349, 0),
					wantTicks: []time.Time{time.Unix(12348, 0)},
				},
				{
					reset:     time.Second,
					wantTime:  time.Unix(12351, 0),
					wantTicks: []time.Time{time.Unix(12350, 0)},
				},
				{
					resetAbsolute: time.Unix(12354, 0),
					wantTime:      time.Unix(12353, 0),
				},
				{
					wantTime:  time.Unix(12355, 0),
					wantTicks: []time.Time{time.Unix(12354, 0)},
				},
			},
		},
		{
			name:  "reset while active",
			start: time.Unix(12345, 0),
			step:  2 * time.Second,
			delay: 3 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
				},
				{
					reset:       3 * time.Second,
					resetReturn: true,
					wantTime:    time.Unix(12349, 0),
				},
				{
					resetAbsolute: time.Unix(12354, 0),
					resetReturn:   true,
					wantTime:      time.Unix(12351, 0),
				},
				{
					wantTime: time.Unix(12353, 0),
				},
				{
					wantTime:  time.Unix(12355, 0),
					wantTicks: []time.Time{time.Unix(12354, 0)},
				},
			},
		},
		{
			name:  "stop after fire",
			start: time.Unix(12345, 0),
			step:  2 * time.Second,
			delay: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime:  time.Unix(12347, 0),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
				{
					stop:     true,
					wantTime: time.Unix(12349, 0),
				},
				{
					wantTime: time.Unix(12351, 0),
				},
				{
					advance:  10 * time.Second,
					wantTime: time.Unix(12361, 0),
				},
			},
		},
		{
			name:  "stop before fire",
			start: time.Unix(12345, 0),
			step:  2 * time.Second,
			delay: time.Second,
			steps: []testStep{
				{
					stop:       true,
					stopReturn: true,
					wantTime:   time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
				},
				{
					wantTime: time.Unix(12349, 0),
				},
				{
					wantTime: time.Unix(12351, 0),
				},
				{
					advance:  10 * time.Second,
					wantTime: time.Unix(12361, 0),
				},
			},
		},
		{
			name:  "stop after reset",
			start: time.Unix(12345, 0),
			step:  2 * time.Second,
			delay: time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime:  time.Unix(12347, 0),
					wantTicks: []time.Time{time.Unix(12346, 0)},
				},
				{
					reset:    10 * time.Second,
					wantTime: time.Unix(12349, 0),
				},
				{
					stop:       true,
					stopReturn: true,
					wantTime:   time.Unix(12351, 0),
				},
				{
					advance:  10 * time.Second,
					wantTime: time.Unix(12361, 0),
				},
			},
		},
		{
			name:  "reset while running",
			start: time.Unix(12345, 0),
			delay: 2 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					advance:  time.Second,
					wantTime: time.Unix(12346, 0),
				},
				{
					advance:  time.Second,
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12347, 0),
					},
				},
				{
					advance:  time.Second,
					reset:    time.Second,
					wantTime: time.Unix(12348, 0),
					wantTicks: []time.Time{
						time.Unix(12348, 0),
					},
				},
				{
					setStep:  5 * time.Second,
					reset:    10 * time.Second,
					wantTime: time.Unix(12353, 0),
				},
				{
					wantTime: time.Unix(12358, 0),
					wantTicks: []time.Time{
						time.Unix(12358, 0),
					},
				},
			},
		},
		{
			name:  "reset while stopped",
			start: time.Unix(12345, 0),
			step:  time.Second,
			delay: 2 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12346, 0),
				},
				{
					stop:       true,
					stopReturn: true,
					wantTime:   time.Unix(12347, 0),
				},
				{
					wantTime: time.Unix(12348, 0),
				},
				{
					wantTime: time.Unix(12349, 0),
				},
				{
					reset:    time.Second,
					wantTime: time.Unix(12350, 0),
					wantTicks: []time.Time{
						time.Unix(12350, 0),
					},
				},
				{
					wantTime: time.Unix(12351, 0),
				},
			},
		},
		{
			name:  "reset absolute",
			start: time.Unix(12345, 0),
			step:  time.Second,
			delay: 2 * time.Second,
			steps: []testStep{
				{
					wantTime: time.Unix(12345, 0),
				},
				{
					wantTime: time.Unix(12346, 0),
				},
				{
					wantTime: time.Unix(12347, 0),
					wantTicks: []time.Time{
						time.Unix(12347, 0),
					},
				},
				{
					resetAbsolute: time.Unix(12354, 50),
					advance:       7 * time.Second,
					wantTime:      time.Unix(12354, 0),
				},
				{
					wantTime: time.Unix(12355, 0),
					wantTicks: []time.Time{
						time.Unix(12354, 50),
					},
				},
				{
					wantTime: time.Unix(12356, 0),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			clock := NewClockBuilder().
				Start(tt.start).
				Step(tt.step).
				Finish()
			tc, tickC := clock.NewTimer(tt.delay)
			timerControl := tc.(*Timer)

			t.Cleanup(func() { timerControl.Stop() })

			expectNoTicks(t, tickC)

			for i, step := range tt.steps {
				if step.stop {
					if got := timerControl.Stop(); got != step.stopReturn {
						t.Errorf("step %v Stop returned %v, want %v", i, got, step.stopReturn)
					}
				}

				if !step.resetAbsolute.IsZero() {
					if got := timerControl.ResetAbsolute(step.resetAbsolute); got != step.resetReturn {
						t.Errorf("step %v Reset returned %v, want %v", i, got, step.resetReturn)
					}
				}

				if step.reset > 0 {
					if got := timerControl.Reset(step.reset); got != step.resetReturn {
						t.Errorf("step %v Reset returned %v, want %v", i, got, step.resetReturn)
					}
				}

				if step.setStep > 0 {
					clock.SetStep(step.setStep)
				}

				if step.advance > 0 {
					clock.Advance(step.advance)
				}

				if now := clock.Now(); !step.wantTime.IsZero() && !now.Equal(step.wantTime) {
					t.Errorf("step %v now = %v, want %v", i, now, step.wantTime)
				}

				for j, want := range step.wantTicks {
					select {
					case tick := <-tickC:
						if tick.Equal(want) {
							continue
						}
						t.Errorf("step %v tick %v = %v, want %v", i, j, tick, want)
					default:
						t.Errorf("step %v tick %v missing", i, j)
					}
				}

				expectNoTicks(t, tickC)
			}
		})
	}
}

type testEvent struct {
	fireTimes     []time.Time
	scheduleTimes []time.Time
}

func (te *testEvent) Fire(t time.Time) time.Time {
	var ret time.Time

	te.fireTimes = append(te.fireTimes, t)
	if len(te.scheduleTimes) > 0 {
		ret = te.scheduleTimes[0]
		te.scheduleTimes = te.scheduleTimes[1:]
	}
	return ret
}

func TestEventManager(t *testing.T) {
	t.Parallel()

	var em eventManager

	testEvents := []testEvent{
		{
			scheduleTimes: []time.Time{
				time.Unix(12300, 0), // step 1
				time.Unix(12340, 0), // step 1
				time.Unix(12345, 0), // step 1
				time.Unix(12346, 0), // step 1
				time.Unix(12347, 0), // step 3
				time.Unix(12348, 0), // step 4
				time.Unix(12349, 0), // step 4
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(12350, 0), // step 4
				time.Unix(12360, 0), // step 5
				time.Unix(12370, 0), // rescheduled
				time.Unix(12380, 0), // step 6
				time.Unix(12381, 0), // step 6
				time.Unix(12382, 0), // step 6
				time.Unix(12393, 0), // stopped
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(12350, 1), // step 4
				time.Unix(12360, 1), // rescheduled
				time.Unix(12370, 1), // step 6
				time.Unix(12380, 1), // step 6
				time.Unix(12381, 1), // step 6
				time.Unix(12382, 1), // step 6
				time.Unix(12383, 1), // step 6
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(12355, 0), // step 5
				time.Unix(12365, 0), // step 5
				time.Unix(12370, 0), // step 6
				time.Unix(12390, 0), // step 6
				time.Unix(12391, 0), // step 7
				time.Unix(12392, 0), // step 7
				time.Unix(12393, 0), // step 7
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(100000, 0), // step 7
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(12346, 0), // step 1
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(12305, 0), // step 5
			},
		},
		{
			scheduleTimes: []time.Time{
				time.Unix(12372, 0), // step 6
				time.Unix(12374, 0), // step 6
				time.Unix(12376, 0), // step 6
				time.Unix(12386, 0), // step 6
				time.Unix(12396, 0), // step 7
			},
		},
	}

	steps := []struct {
		reschedule    []int
		stop          []int
		advanceTo     time.Time
		want          map[int][]time.Time
		waitingEvents int
	}{
		{
			advanceTo: time.Unix(12345, 0),
		},
		{
			reschedule: []int{0, 1, 2, 3, 4, 5}, // add 0, 1, 2, 3, 4, 5
			advanceTo:  time.Unix(12346, 0),
			want: map[int][]time.Time{
				0: {
					time.Unix(12300, 0),
					time.Unix(12340, 0),
					time.Unix(12345, 0),
					time.Unix(12346, 0),
				},
				5: {
					time.Unix(12346, 0),
				},
			},
			waitingEvents: 5, // scheduled 0, 1, 2, 3, 4, 5; retired 5
		},
		{
			advanceTo:     time.Unix(12346, 50),
			waitingEvents: 5, // no change
		},
		{
			advanceTo: time.Unix(12347, 50),
			want: map[int][]time.Time{
				0: {
					time.Unix(12347, 0),
				},
			},
			waitingEvents: 5, // no change
		},
		{
			advanceTo: time.Unix(12350, 50),
			want: map[int][]time.Time{
				0: {
					time.Unix(12348, 0),
					time.Unix(12349, 0),
				},
				1: {
					time.Unix(12350, 0),
				},
				2: {
					time.Unix(12350, 1),
				},
			},
			waitingEvents: 4, // retired 0
		},
		{
			reschedule: []int{6, 7}, // add 6, 7
			stop:       []int{2},
			advanceTo:  time.Unix(12365, 0),
			want: map[int][]time.Time{
				1: {
					time.Unix(12360, 0),
				},
				3: {
					time.Unix(12355, 0),
					time.Unix(12365, 0),
				},
				6: {
					time.Unix(12305, 0),
				},
			},
			waitingEvents: 4, // scheduled 6, 7; retired 2, 5
		},
		{
			reschedule: []int{1, 2}, // update 1; add 2
			stop:       []int{6},
			advanceTo:  time.Unix(12390, 0),
			want: map[int][]time.Time{
				1: {
					time.Unix(12380, 0),
					time.Unix(12381, 0),
					time.Unix(12382, 0),
				},
				2: {
					time.Unix(12370, 1),
					time.Unix(12380, 1),
					time.Unix(12381, 1),
					time.Unix(12382, 1),
					time.Unix(12383, 1),
				},
				3: {
					time.Unix(12370, 0),
					time.Unix(12390, 0),
				},
				7: {
					time.Unix(12372, 0),
					time.Unix(12374, 0),
					time.Unix(12376, 0),
					time.Unix(12386, 0),
				},
			},
			waitingEvents: 3, // scheduled 2, retired 2, stopped 6
		},
		{
			stop:      []int{1}, // no-op: already stopped
			advanceTo: time.Unix(200000, 0),
			want: map[int][]time.Time{
				3: {
					time.Unix(12391, 0),
					time.Unix(12392, 0),
					time.Unix(12393, 0),
				},
				4: {
					time.Unix(100000, 0),
				},
				7: {
					time.Unix(12396, 0),
				},
			},
			waitingEvents: 0, // retired 3, 4, 7
		},
		{
			advanceTo: time.Unix(300000, 0),
		},
	}

	for i, step := range steps {
		for _, idx := range step.reschedule {
			ev := &testEvents[idx]
			t := ev.scheduleTimes[0]
			ev.scheduleTimes = ev.scheduleTimes[1:]
			em.Reschedule(ev, t)
		}
		for _, idx := range step.stop {
			ev := &testEvents[idx]
			em.Reschedule(ev, time.Time{})
		}
		em.AdvanceTo(step.advanceTo)
		for j := range testEvents {
			if !slices.Equal(testEvents[j].fireTimes, step.want[j]) {
				t.Errorf("step %v event %v fire times = %v, want %v", i, j, testEvents[j].fireTimes, step.want[j])
			}
			testEvents[j].fireTimes = nil
		}
	}
}
