package main

import (
	"context"
	"reflect"
	"time"

	"th10/trace"
)

type sampleOptions struct {
	Settle, Gap, Budget, Poll time.Duration
	Attempts                  int
}

type sampleClock struct {
	Now   func() time.Time
	Sleep func(time.Duration)
}

func realClock() sampleClock { return sampleClock{Now: time.Now, Sleep: time.Sleep} }

type sampleResult struct {
	Frame    *trace.Frame
	Attempts int
	Elapsed  time.Duration
	Err      error
	Control  control
}

// sampleSettled never silently drops a requested frame. A completion witness
// is needed even if two reads would agree during a long GDI calc stall.
// The budget bounds waits/retries, not an in-flight ReadProcessMemory call.
func (r *reader) sampleSettled(ctx context.Context, first control, opts sampleOptions, clock sampleClock) (res sampleResult) {
	start := clock.Now()
	deadline, settled := start.Add(opts.Budget), start.Add(opts.Settle)
	last := first
	reason := "calc-unconfirmed"
	var previous *snapshot
	res.Frame = &trace.Frame{Index: int(first.Frame)}
	defer func() {
		res.Elapsed = clock.Now().Sub(start)
		res.Control = last
		if reason != "" {
			res.Frame.Half = &trace.HalfRow{Reason: reason, Attempts: res.Attempts,
				Before: int(first.Frame), After: int(last.Frame)}
		}
	}()
	complete := calcWitness(first, first)
	for clock.Now().Before(deadline) {
		if ctx.Err() != nil {
			reason = "interrupted"
			res.Err = ctx.Err()
			return res
		}
		before, err := r.readControl()
		if err != nil {
			reason, res.Err = "read-error", err
			return res
		}
		last = before
		if !first.sameFrame(before) {
			reason = "cross-frame"
			return res
		}
		complete = complete || calcWitness(first, before)
		if !complete || clock.Now().Before(settled) {
			clock.Sleep(min(opts.Poll, deadline.Sub(clock.Now())))
			continue
		}
		res.Attempts++
		current, err := r.sampleFrame(before)
		if current != nil {
			res.Frame = current.Frame
		}
		after, guardErr := r.readControl()
		if guardErr != nil {
			reason, res.Err = "read-error", guardErr
			return res
		}
		last = after
		if !first.sameFrame(after) {
			reason = "cross-frame"
			return res
		}
		if err == nil && previous != nil && reflect.DeepEqual(previous, current) {
			reason, res.Err = "", nil
			return res
		}
		if err != nil {
			previous, reason, res.Err = nil, "read-error", err
		} else {
			previous, reason, res.Err = current, "unstable", nil
		}
		if res.Attempts >= opts.Attempts || !clock.Now().Before(deadline) {
			return res
		}
		clock.Sleep(min(opts.Gap, deadline.Sub(clock.Now())))
	}
	return res
}
