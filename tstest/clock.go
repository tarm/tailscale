// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package tstest

import (
	"container/heap"
	"sync"
	"time"

	"tailscale.com/tstime"
	"tailscale.com/util/mak"
)

// ClockBuilder is used to configure the initial settings for a Clock. Once
// the settings are configured as desired, call ClockBuilder.Finish to get the
// resulting Clock. ClockBuilder can only be used to create a single Clock.
type ClockBuilder struct {
	c *Clock
}

func NewClockBuilder() *ClockBuilder {
	cb := &ClockBuilder{
		c: new(Clock),
	}
	return cb
}

func (cb *ClockBuilder) Start(t time.Time) *ClockBuilder {
	cb.c.start = t
	return cb
}

func (cb *ClockBuilder) Step(d time.Duration) *ClockBuilder {
	cb.c.step = d
	return cb
}

func (cb *ClockBuilder) TimerChannelSize(n int) *ClockBuilder {
	cb.c.timerChannelSize = n
	return cb
}

func (cb *ClockBuilder) Finish() *Clock {
	c := cb.c
	cb.c = nil
	c.init()
	return c
}

// NewClock returns a Clock initialized with default settings (equivalent to
// NewClockBuilder().Finish()).
func NewClock() *Clock {
	c := new(Clock)
	c.init()
	return c
}

// Clock is a testing clock that advances every time its Now method is
// called, beginning at its start time. If no start time is specified using
// ClockBuilder, an arbitrary start time will be selected when the Clock is
// created and can be retrieved by calling Clock.Start().
type Clock struct {
	// start is the first value returned by Now. It must not be modified after
	// init() is called.
	start time.Time

	mu sync.Mutex

	// step is how much to advance with each Now call.
	step time.Duration
	// present is the last value returned by Now (and will be returned again by
	// ConstantNow).
	present time.Time
	// skipStep indicates that the next call to Now should not add step to
	// present. This occurs after initialization and after Advance.
	skipStep bool
	// timerChannelSize is the buffer size to use for channels created by
	// NewTimer and NewTicker.
	timerChannelSize int

	events eventManager
}

func (c *Clock) init() {
	if c.start.IsZero() {
		c.start = time.Now()
	}
	if c.timerChannelSize == 0 {
		c.timerChannelSize = 1
	}
	c.present = c.start
	c.skipStep = true
	c.events.AdvanceTo(c.present)
}

// Now returns the virtual clock's current time, and advances it
// according to its step configuration.
func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.skipStep && c.step > 0 {
		c.present = c.present.Add(c.step)
		c.events.AdvanceTo(c.present)
	}
	c.skipStep = false
	return c.present
}

// ConstantNow returns the virtual clock's current time but does not advance it
// if a step is configured.
func (c *Clock) ConstantNow() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.present
}

func (c *Clock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.skipStep = true

	if d > 0 {
		c.present = c.present.Add(d)
		c.events.AdvanceTo(c.present)
	}

	return c.present
}

func (c *Clock) Start() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.start
}

func (c *Clock) Step() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.step
}

func (c *Clock) SetStep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.step = d
}

func (c *Clock) SetTimerChannelSize(n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timerChannelSize = n
}

// NewTicker returns a Ticker that uses this Clock for accessing the current
// time.
func (c *Clock) NewTicker(d time.Duration) (tstime.TickerController, <-chan time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := &Ticker{
		nextTrigger: c.present.Add(d),
		period:      d,
		em:          &c.events,
	}
	t.init(c.timerChannelSize)
	return t, t.C
}

// NewTimer returns a Timer that uses this Clock for accessing the current
// time.
func (c *Clock) NewTimer(d time.Duration) (tstime.TimerController, <-chan time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := &Timer{
		nextTrigger: c.present.Add(d),
		em:          &c.events,
	}
	t.init(c.timerChannelSize)
	return t, t.C
}

type eventHandler interface {
	// Fire signals the event. The provided time is written to the event's
	// channel as the current time. The return value is the next time this event
	// should fire, otherwise if it is zero then the event will be removed from
	// the eventManager.
	Fire(time.Time) time.Time
}

type event struct {
	position int       // The current index in the heap.
	when     time.Time // A cache of the next time the event triggers.
	eh       eventHandler
}

// eventManager tracks pending events created by Timer and Ticker. eventManager
// implements heap.Interface for efficient lookups of the next event.
type eventManager struct {
	mu            sync.Mutex
	now           time.Time
	heap          []*event
	reverseLookup map[eventHandler]*event
}

// Push implements heap.Interface.Push and must only be called by heap funcs
// with em.mu already held.
func (em *eventManager) Push(x any) {
	e, ok := x.(*event)
	if !ok {
		panic("incorrect event type")
	}
	if e == nil {
		panic("nil event")
	}

	mak.Set(&em.reverseLookup, e.eh, e)
	e.position = len(em.heap)
	em.heap = append(em.heap, e)
}

// Pop implements heap.Interface.Pop and must only be called by heap funcs with
// em.mu already held.
func (em *eventManager) Pop() any {
	e := em.heap[len(em.heap)-1]
	em.heap = em.heap[:len(em.heap)-1]
	delete(em.reverseLookup, e.eh)
	return e
}

// Len implements sort.Interface.Len and must only be called by heap funcs with
// em.mu already held.
func (em *eventManager) Len() int {
	return len(em.heap)
}

// Less implements sort.Interface.Less and must only be called by heap funcs
// with em.mu already held.
func (em *eventManager) Less(i, j int) bool {
	return em.heap[i].when.Before(em.heap[j].when)
}

// Swap implements sort.Interface.Swap and must only be called by heap funcs
// with em.mu already held.
func (em *eventManager) Swap(i, j int) {
	em.heap[i], em.heap[j] = em.heap[j], em.heap[i]
	em.heap[i].position = i
	em.heap[j].position = j
}

// Reschedule adds/updates/deletes an event in the heap, whichever
// operation is applicable (use a zero time to delete).
func (em *eventManager) Reschedule(eh eventHandler, t time.Time) {
	em.mu.Lock()
	defer em.mu.Unlock()
	e, ok := em.reverseLookup[eh]
	if !ok {
		if t.IsZero() {
			// eh is not scheduled and also not active, so do nothing.
			return
		}
		// eh is not scheduled but is active, so add it.
		heap.Push(em, &event{
			when: t,
			eh:   eh,
		})
		return
	}

	if t.IsZero() {
		// e is scheduled but not active, so remove it.
		heap.Remove(em, e.position)
		return
	}

	// e is scheduled and active, so update it.
	e.when = t
	heap.Fix(em, e.position)
}

func (em *eventManager) AdvanceTo(tm time.Time) {
	for len(em.heap) > 0 {
		if em.heap[0].when.After(tm) {
			break
		}

		// Ideally some jitter would be added here but it's difficult to do so
		// in a deterministic fashion.
		em.now = em.heap[0].when

		if nextFire := em.heap[0].eh.Fire(em.now); !nextFire.IsZero() {
			em.heap[0].when = nextFire
			heap.Fix(em, 0)
		} else {
			heap.Pop(em)
		}
	}

	em.now = tm
}

func (em *eventManager) Now() time.Time {
	em.mu.Lock()
	defer em.mu.Unlock()
	return em.now
}

// Ticker is a time.Ticker lookalike for use in tests that need to control when
// events fire. Ticker could be made standalone in future but for now is
// expected to be paired with a Clock and created by Clock.NewTicker.
type Ticker struct {
	C <-chan time.Time // The channel on which ticks are delivered.

	// em is the eventManager to be notified when nextTrigger changes.
	// eventManager has its own mutex, and the pointer is immutable, therefore
	// em can be accessed without holding mu.
	em *eventManager

	c chan<- time.Time // The writer side of C.

	mu sync.Mutex

	// nextTrigger is the time of the ticker's next scheduled activation. When
	// Fire activates the ticker, nextTrigger is the timestamp written to the
	// channel.
	nextTrigger time.Time

	// period is the duration that is added to nextTrigger when the ticker
	// fires.
	period time.Duration
}

func (t *Ticker) init(channelSize int) {
	if channelSize <= 0 {
		panic("ticker channel size must be non-negative")
	}
	c := make(chan time.Time, channelSize)
	t.c = c
	t.C = c
	t.em.Reschedule(t, t.nextTrigger)
}

// Fire triggers the ticker. curTime is the timestamp to write to the channel.
// The next trigger time for the ticker is updated to the last computed trigger
// time + the ticker period (set at creation or using Reset). The next trigger
// time is computed this way to match standard time.Ticker behavior, which
// prevents accumulation of long term drift caused by delays in event execution.
func (t *Ticker) Fire(curTime time.Time) time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.nextTrigger.IsZero() {
		return t.nextTrigger
	}
	select {
	case t.c <- curTime:
	default:
	}
	t.nextTrigger = t.nextTrigger.Add(t.period)

	return t.nextTrigger
}

// Reset adjusts the Ticker's period to d and reschedules the next fire time to
// the current simulated time + d.
func (t *Ticker) Reset(d time.Duration) {
	if d <= 0 {
		// The standard time.Ticker requires a positive period.
		panic("non-positive period for Ticker.Reset")
	}

	now := t.em.Now()

	t.mu.Lock()
	t.resetLocked(now.Add(d), d)
	t.mu.Unlock()

	t.em.Reschedule(t, t.nextTrigger)
}

// ResetAbsolute adjusts the Ticker's period to d and reschedules the next fire
// time to nextTrigger.
func (t *Ticker) ResetAbsolute(nextTrigger time.Time, d time.Duration) {
	if nextTrigger.IsZero() {
		panic("zero nextTrigger time for ResetAbsolute")
	}
	if d <= 0 {
		panic("non-positive period for ResetAbsolute")
	}

	t.mu.Lock()
	t.resetLocked(nextTrigger, d)
	t.mu.Unlock()

	t.em.Reschedule(t, t.nextTrigger)
}

func (t *Ticker) resetLocked(nextTrigger time.Time, d time.Duration) {
	t.nextTrigger = nextTrigger
	t.period = d
}

func (t *Ticker) Stop() {
	t.mu.Lock()
	t.nextTrigger = time.Time{}
	t.mu.Unlock()

	t.em.Reschedule(t, t.nextTrigger)
}

// Timer is a time.Timer lookalike for use in tests that need to control when
// events fire. Timer could be made standalone in future but for now must be
// paired with a Clock and created by Clock.NewTimer.
type Timer struct {
	C <-chan time.Time // The channel on which ticks are delivered.

	// em is the eventManager to be notified when nextTrigger changes.
	// eventManager has its own mutex, and the pointer is immutable, therefore
	// em can be accessed without holding mu.
	em *eventManager

	c chan<- time.Time // The writer side of C.

	mu sync.Mutex

	// nextTrigger is the time of the ticker's next scheduled activation. When
	// Fire activates the ticker, nextTrigger is the timestamp written to the
	// channel.
	nextTrigger time.Time
}

func (t *Timer) init(channelSize int) {
	if channelSize <= 0 {
		panic("ticker channel size must be non-negative")
	}
	c := make(chan time.Time, channelSize)
	t.c = c
	t.C = c
	t.em.Reschedule(t, t.nextTrigger)
}

// Fire triggers the ticker. curTime is the timestamp to write to the channel.
// The next trigger time for the ticker is updated to the last computed trigger
// time + the ticker period (set at creation or using Reset). The next trigger
// time is computed this way to match standard time.Ticker behavior, which
// prevents accumulation of long term drift caused by delays in event execution.
func (t *Timer) Fire(curTime time.Time) time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.nextTrigger.IsZero() {
		return time.Time{}
	}
	t.nextTrigger = time.Time{}
	select {
	case t.c <- curTime:
	default:
	}
	return time.Time{}
}

// Reset reschedules the next fire time to the current simulated time + d.
// Reset returns true if the timer was still active before the reset.
func (t *Timer) Reset(d time.Duration) bool {
	if d <= 0 {
		// The standard time.Timer requires a positive delay.
		panic("non-positive delay for Timer.Reset")
	}

	return t.reset(t.em.Now().Add(d))
}

// ResetAbsolute reschedules the next fire time to nextTrigger.
// ResetAbsolute returns true if the timer was still active before the reset.
func (t *Timer) ResetAbsolute(nextTrigger time.Time) bool {
	if nextTrigger.IsZero() {
		panic("zero nextTrigger time for ResetAbsolute")
	}

	return t.reset(nextTrigger)
}

// Stop deactivates the Timer. Stop returns true if the timer was active before
// stopping.
func (t *Timer) Stop() bool {
	return t.reset(time.Time{})
}

func (t *Timer) reset(nextTrigger time.Time) bool {
	t.mu.Lock()
	wasActive := !t.nextTrigger.IsZero()
	t.nextTrigger = nextTrigger
	t.mu.Unlock()

	t.em.Reschedule(t, t.nextTrigger)
	return wasActive
}
