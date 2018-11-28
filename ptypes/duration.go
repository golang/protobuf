// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptypes

// This file implements conversions between google.protobuf.Duration
// and time.Duration.

import (
	"errors"
	"fmt"
	"time"

	durpb "github.com/golang/protobuf/ptypes/duration"
)

const (
	// Range of a durpb.Duration in seconds, as specified in
	// google/protobuf/duration.proto. This is about 10,000 years in seconds.
	maxSeconds = int64(10000 * 365.25 * 24 * 60 * 60)
	minSeconds = -maxSeconds
)

// validateDuration determines whether the durpb.Duration is valid according to the
// definition in google/protobuf/duration.proto. A valid durpb.Duration
// may still be too large to fit into a time.Duration (the range of durpb.Duration
// is about 10,000 years, and the range of time.Duration is about 290).
func validateDuration(d *durpb.Duration) error {
	if d == nil {
		return errors.New("duration: nil Duration")
	}
	if d.Seconds < minSeconds || d.Seconds > maxSeconds {
		return fmt.Errorf("duration: %v: seconds out of range", d)
	}
	if d.Nanos <= -1e9 || d.Nanos >= 1e9 {
		return fmt.Errorf("duration: %v: nanos out of range", d)
	}
	// Seconds and Nanos must have the same sign, unless d.Nanos is zero.
	if (d.Seconds < 0 && d.Nanos > 0) || (d.Seconds > 0 && d.Nanos < 0) {
		return fmt.Errorf("duration: %v: seconds and nanos have different signs", d)
	}
	return nil
}

// Duration converts a durpb.Duration to a time.Duration. Duration
// returns an error if the durpb.Duration is invalid or is too large to be
// represented in a time.Duration.
func Duration(p *durpb.Duration) (time.Duration, error) {
	if err := validateDuration(p); err != nil {
		return 0, err
	}
	d := time.Duration(p.Seconds) * time.Second
	if int64(d/time.Second) != p.Seconds {
		return 0, fmt.Errorf("duration: %v is out of range for time.Duration", p)
	}
	if p.Nanos != 0 {
		d += time.Duration(p.Nanos)
		if (d < 0) != (p.Nanos < 0) {
			return 0, fmt.Errorf("duration: %v is out of range for time.Duration", p)
		}
	}
	return d, nil
}

// DurationProto converts a time.Duration to a durpb.Duration.
func DurationProto(d time.Duration) *durpb.Duration {
	nanos := d.Nanoseconds()
	secs := nanos / 1e9
	nanos -= secs * 1e9
	return &durpb.Duration{
		Seconds: secs,
		Nanos:   int32(nanos),
	}
}
