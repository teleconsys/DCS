// Package service hosts the GUI's adapters over `internal/*`: it owns the
// async runner that funnels long-running work into goroutines, captures
// output for the UI, and builds the param structs that `internal/cid`,
// `internal/offers`, `internal/whitelist`, etc. consume.
package service

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// Job is a unit of work submitted to the Runner.
//
// The Fn is invoked on a background goroutine. It receives the same
// `out` writer passed to Run; everything written to `out` is appended to
// the view that owns the writer (the UI keeps the writer thread-safe).
type Job struct {
	Title string
	Fn    func(ctx context.Context, out io.Writer) error
}

// Runner serializes Job execution per view: while one job is running, a
// subsequent Run call refuses to start a second one. This mirrors the
// "Run" button being disabled while an action is in flight.
type Runner struct {
	mu      sync.Mutex
	running bool
}

// NewRunner returns a fresh runner.
func NewRunner() *Runner { return &Runner{} }

// Busy reports whether a job is currently in flight.
func (r *Runner) Busy() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Run starts the job in a goroutine. The `onDone` callback (if non-nil)
// is invoked with the final error after Fn returns; callers typically
// schedule this onto the UI thread themselves via fyne.Do.
//
// Run returns immediately. If the runner is already busy, Run is a no-op
// and onDone is invoked with an error.
func (r *Runner) Run(ctx context.Context, out io.Writer, job Job, onDone func(error)) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		if onDone != nil {
			onDone(fmt.Errorf("another action is already running"))
		}
		return
	}
	r.running = true
	r.mu.Unlock()

	go func() {
		start := time.Now()
		title := job.Title
		if title == "" {
			title = "action"
		}
		fmt.Fprintf(out, "▶ %s\n", title)
		err := safeRun(ctx, out, job.Fn)
		if err != nil {
			fmt.Fprintf(out, "✗ %s failed after %s: %v\n\n", title, time.Since(start).Round(time.Millisecond), err)
		} else {
			fmt.Fprintf(out, "✓ %s ok (%s)\n\n", title, time.Since(start).Round(time.Millisecond))
		}

		r.mu.Lock()
		r.running = false
		r.mu.Unlock()

		if onDone != nil {
			onDone(err)
		}
	}()
}

// safeRun protects the goroutine from panics inside Fn so the runner
// always releases its lock and the UI always gets a final callback.
func safeRun(ctx context.Context, out io.Writer, fn func(ctx context.Context, out io.Writer) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
			fmt.Fprintf(out, "PANIC: %v\n", r)
		}
	}()
	return fn(ctx, out)
}
