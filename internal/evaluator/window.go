package evaluator

import (
	"fmt"
	"time"
)

// WindowCheck determines if we're currently within a time window,
// and the boundaries of the current/next window.
type WindowCheck struct {
	InWindow    bool
	WindowStart time.Time // start of current (or most recent) window
	WindowEnd   time.Time // end of current (or most recent) window
}

// CheckWindow evaluates whether the current time falls within a daily time window.
// If windowStart/windowEnd are empty, returns InWindow=true (all day).
func CheckWindow(windowStart, windowEnd, tz string, now time.Time) (WindowCheck, error) {
	if windowStart == "" || windowEnd == "" {
		return WindowCheck{InWindow: true}, nil
	}

	loc := time.UTC
	if tz != "" {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			return WindowCheck{}, fmt.Errorf("invalid timezone %q: %w", tz, err)
		}
	}

	localNow := now.In(loc)

	ws, err := parseTimeOfDay(windowStart, localNow, loc)
	if err != nil {
		return WindowCheck{}, fmt.Errorf("parse window_start: %w", err)
	}

	we, err := parseTimeOfDay(windowEnd, localNow, loc)
	if err != nil {
		return WindowCheck{}, fmt.Errorf("parse window_end: %w", err)
	}

	inWindow := !localNow.Before(ws) && !localNow.After(we)

	return WindowCheck{
		InWindow:    inWindow,
		WindowStart: ws,
		WindowEnd:   we,
	}, nil
}

// parseTimeOfDay parses "HH:MM" into today's date in the given location.
func parseTimeOfDay(s string, today time.Time, loc *time.Location) (time.Time, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return time.Time{}, fmt.Errorf("expected HH:MM format, got %q", s)
	}
	return time.Date(today.Year(), today.Month(), today.Day(), h, m, 0, 0, loc), nil
}
