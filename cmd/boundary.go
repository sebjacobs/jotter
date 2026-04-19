package cmd

import (
	"fmt"
	"time"

	"github.com/sebjacobs/jotter/internal"
)

// parseBoundary parses either a date (YYYY-MM-DD) or full timestamp
// (YYYY-MM-DDTHH:MM:SS). For date-only values, endOfDay=true promotes
// the result to 23:59:59 so --until <date> is inclusive of that day.
func parseBoundary(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(internal.TimestampFormat, s); err == nil {
		return t, nil
	}
	t, err := time.Parse(internal.DateFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS, got %q", s)
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t, nil
}

// parseWindow reads --since / --until flags and returns the parsed bounds,
// validating that until is not earlier than since.
func parseWindow(since, until string) (sinceTime, untilTime time.Time, err error) {
	if since != "" {
		sinceTime, err = parseBoundary(since, false)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --since value: %w", err)
		}
	}
	if until != "" {
		untilTime, err = parseBoundary(until, true)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --until value: %w", err)
		}
	}
	if !sinceTime.IsZero() && !untilTime.IsZero() && untilTime.Before(sinceTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("--until must not be earlier than --since")
	}
	return sinceTime, untilTime, nil
}

// inWindow reports whether the entry timestamp falls within [since, until].
// Zero-value bounds are treated as open (no constraint on that side).
func inWindow(entryTS string, since, until time.Time) bool {
	if since.IsZero() && until.IsZero() {
		return true
	}
	t, err := time.Parse(internal.TimestampFormat, entryTS)
	if err != nil {
		return false
	}
	if !since.IsZero() && t.Before(since) {
		return false
	}
	if !until.IsZero() && t.After(until) {
		return false
	}
	return true
}
