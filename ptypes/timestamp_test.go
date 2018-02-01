// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2016 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package ptypes

import (
	"math"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
)

const (
	// The max value of time.Duration
	maxDuration = 1<<63 - 1
	// The min value of time.Duration
	minDuration = -1 << 63
)

var tests = []struct {
	ts    *tspb.Timestamp
	valid bool
	t     time.Time
}{
	// The timestamp representing the Unix epoch date.
	{&tspb.Timestamp{Seconds: 0, Nanos: 0}, true, utcDate(1970, 1, 1)},
	// The smallest representable timestamp.
	{&tspb.Timestamp{Seconds: math.MinInt64, Nanos: math.MinInt32}, false,
		time.Unix(math.MinInt64, math.MinInt32).UTC()},
	// The smallest representable timestamp with non-negative nanos.
	{&tspb.Timestamp{Seconds: math.MinInt64, Nanos: 0}, false, time.Unix(math.MinInt64, 0).UTC()},
	// The earliest valid timestamp.
	{&tspb.Timestamp{Seconds: minValidSeconds, Nanos: 0}, true, utcDate(1, 1, 1)},
	//"0001-01-01T00:00:00Z"},
	// The largest representable timestamp.
	{&tspb.Timestamp{Seconds: math.MaxInt64, Nanos: math.MaxInt32}, false,
		time.Unix(math.MaxInt64, math.MaxInt32).UTC()},
	// The largest representable timestamp with nanos in range.
	{&tspb.Timestamp{Seconds: math.MaxInt64, Nanos: 1e9 - 1}, false,
		time.Unix(math.MaxInt64, 1e9-1).UTC()},
	// The largest valid timestamp.
	{&tspb.Timestamp{Seconds: maxValidSeconds - 1, Nanos: 1e9 - 1}, true,
		time.Date(9999, 12, 31, 23, 59, 59, 1e9-1, time.UTC)},
	// The smallest invalid timestamp that is larger than the valid range.
	{&tspb.Timestamp{Seconds: maxValidSeconds, Nanos: 0}, false, time.Unix(maxValidSeconds, 0).UTC()},
	// A date before the epoch.
	{&tspb.Timestamp{Seconds: -281836800, Nanos: 0}, true, utcDate(1961, 1, 26)},
	// A date after the epoch.
	{&tspb.Timestamp{Seconds: 1296000000, Nanos: 0}, true, utcDate(2011, 1, 26)},
	// A date after the epoch, in the middle of the day.
	{&tspb.Timestamp{Seconds: 1296012345, Nanos: 940483}, true,
		time.Date(2011, 1, 26, 3, 25, 45, 940483, time.UTC)},
}

func TestValidateTimestamp(t *testing.T) {
	for _, s := range tests {
		got := validateTimestamp(s.ts)
		if (got == nil) != s.valid {
			t.Errorf("validateTimestamp(%v) = %v, want %v", s.ts, got, s.valid)
		}
	}
}

func TestTimestamp(t *testing.T) {
	for _, s := range tests {
		got, err := Timestamp(s.ts)
		if (err == nil) != s.valid {
			t.Errorf("Timestamp(%v) error = %v, but valid = %t", s.ts, err, s.valid)
		} else if s.valid && got != s.t {
			t.Errorf("Timestamp(%v) = %v, want %v", s.ts, got, s.t)
		}
	}
	// Special case: a nil Timestamp is an error, but returns the 0 Unix time.
	got, err := Timestamp(nil)
	want := time.Unix(0, 0).UTC()
	if got != want {
		t.Errorf("Timestamp(nil) = %v, want %v", got, want)
	}
	if err == nil {
		t.Errorf("Timestamp(nil) error = nil, expected error")
	}
}

func TestTimestampProto(t *testing.T) {
	for _, s := range tests {
		got, err := TimestampProto(s.t)
		if (err == nil) != s.valid {
			t.Errorf("TimestampProto(%v) error = %v, but valid = %t", s.t, err, s.valid)
		} else if s.valid && !proto.Equal(got, s.ts) {
			t.Errorf("TimestampProto(%v) = %v, want %v", s.t, got, s.ts)
		}
	}
	// No corresponding special case here: no time.Time results in a nil Timestamp.
}

func TestTimestampString(t *testing.T) {
	for _, test := range []struct {
		ts   *tspb.Timestamp
		want string
	}{
		// Not much testing needed because presumably time.Format is
		// well-tested.
		{&tspb.Timestamp{Seconds: 0, Nanos: 0}, "1970-01-01T00:00:00Z"},
		{&tspb.Timestamp{Seconds: minValidSeconds - 1, Nanos: 0}, "(timestamp: seconds:-62135596801  before 0001-01-01)"},
	} {
		got := TimestampString(test.ts)
		if got != test.want {
			t.Errorf("TimestampString(%v) = %q, want %q", test.ts, got, test.want)
		}
	}
}

func utcDate(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestTimestampNow(t *testing.T) {
	// Bracket the expected time.
	before := time.Now()
	ts := TimestampNow()
	after := time.Now()

	tm, err := Timestamp(ts)
	if err != nil {
		t.Errorf("between %v and %v\nTimestampNow() = %v\nwhich is invalid (%v)", before, after, ts, err)
	}
	if tm.Before(before) || tm.After(after) {
		t.Errorf("between %v and %v\nTimestamp(TimestampNow()) = %v", before, after, tm)
	}
}

func TestTimestampAdd(t *testing.T) {
	for _, test := range []struct {
		ts    *tspb.Timestamp
		d     time.Duration
		want  *tspb.Timestamp
		valid bool
	}{
		{
			&tspb.Timestamp{0, 0},
			time.Duration(maxDuration),
			&tspb.Timestamp{maxDuration / int64(time.Second), int32(maxDuration % int64(time.Second))}, true,
		},
		{
			&tspb.Timestamp{0, 0},
			time.Duration(minDuration),
			&tspb.Timestamp{minDuration/int64(time.Second) - 1, int32(minDuration%int64(time.Second)) + 1e9}, true,
		},
		{
			&tspb.Timestamp{maxValidSeconds - 1, 1e9 - 1},
			time.Duration(maxDuration),
			&tspb.Timestamp{maxValidSeconds + maxDuration/int64(time.Second), int32(maxDuration%int64(time.Second)) - 1}, false,
		},
	} {
		tsInTime, _ := Timestamp(test.ts)
		got, err := TimestampAdd(test.ts, test.d)
		if (err == nil) != test.valid {
			t.Errorf("TimestampAdd(%v, %v) = %q\nwhich is invalid (%v)", test.ts, test.d, got, err)
		}
		gotInTime, _ := Timestamp(got)
		addInTime := tsInTime.Add(test.d)
		if !addInTime.Equal(gotInTime) {
			t.Errorf("TimestampAdd(%v, %v) = %q does not match with time.Add(%v, %v) = %q", test.ts, test.d, gotInTime, tsInTime, test.d, addInTime)
		}
		if *got != *test.want {
			t.Errorf("TimestampAdd(%v, %v) = %q, want %q", test.ts, test.d, got, test.want)
		}
	}
}

func TestTimestampSub(t *testing.T) {
	for _, test := range []struct {
		ts    *tspb.Timestamp
		uts   *tspb.Timestamp
		want  time.Duration
		valid bool
	}{
		{
			&tspb.Timestamp{maxDuration / int64(time.Second), int32(maxDuration % int64(time.Second))},
			&tspb.Timestamp{0, 0}, time.Duration(maxDuration),
			true,
		},
		{
			&tspb.Timestamp{minDuration / int64(time.Second), int32(minDuration % int64(time.Second))},
			&tspb.Timestamp{0, 0}, time.Duration(minDuration),
			true,
		},
		{
			&tspb.Timestamp{0, 0},
			&tspb.Timestamp{(minDuration + 1) / int64(time.Second), int32((minDuration + 1) % int64(time.Second))}, time.Duration((-minDuration - 1)),
			true,
		},
		{
			&tspb.Timestamp{maxValidSeconds - 1, 1e9 - 1},
			&tspb.Timestamp{0, 0}, time.Duration(0),
			false,
		},
	} {
		got, err := TimestampSub(test.ts, test.uts)
		if (err == nil) != test.valid {
			t.Errorf("TimestampSub(%v, %v) = %q\nwhich is invalid (%v)", test.ts, test.uts, got, err)
			continue
		}
		if err != nil {
			continue
		}
		if got != test.want {
			t.Errorf("TimestampSub(%v, %v) = %q, want %q", test.ts, test.uts, got, test.want)
		}
		tsInTime, _ := Timestamp(test.ts)
		utsInTime, _ := Timestamp(test.uts)
		gotFromTimes := tsInTime.Sub(utsInTime)
		if got != gotFromTimes {
			t.Errorf("TimestampSub(%v, %v) = %q does not match with time.Sub(%v, %v) = %q", test.ts, test.uts, got, tsInTime, utsInTime, gotFromTimes)
		}
	}
}

func TestTimestampSince(t *testing.T) {
	for _, test := range []struct {
		ts    *tspb.Timestamp
		valid bool
	}{
		{&tspb.Timestamp{minValidSeconds, 0}, false},
		{&tspb.Timestamp{minDuration/int64(time.Second) - 1, int32(minDuration%int64(time.Second)) + 1e9}, false},
		{&tspb.Timestamp{0, 0}, true},
		{&tspb.Timestamp{maxDuration / int64(time.Second), int32(maxDuration % int64(time.Second))}, true},
		{&tspb.Timestamp{maxValidSeconds - 1, 1e9 - 1}, false},
	} {
		testInTime, _ := Timestamp(test.ts)
		before := time.Since(testInTime)
		d, err := TimestampSince(test.ts)
		if (err == nil) != test.valid {
			t.Errorf("TimestampSince(%v) = %q\nwhich is invalid(%v)", test.ts, d, err)
			continue
		}
		if err != nil {
			continue
		}
		after := time.Since(testInTime)
		if d < before || after < d {
			t.Errorf("between %v and %v\nTimestampSince(%v) = %q", before, after, test.ts, d)
		}
	}
}

func TestTimestampUntil(t *testing.T) {
	for _, test := range []struct {
		ts    *tspb.Timestamp
		valid bool
	}{
		{&tspb.Timestamp{minValidSeconds, 0}, false},
		{&tspb.Timestamp{minDuration/int64(time.Second) - 1, int32(minDuration%int64(time.Second)) + 1e9}, false},
		{&tspb.Timestamp{0, 0}, true},
		{&tspb.Timestamp{maxDuration / int64(time.Second), int32(maxDuration % int64(time.Second))}, true},
		{&tspb.Timestamp{maxValidSeconds - 1, 1e9 - 1}, false},
	} {
		testInTime, _ := Timestamp(test.ts)
		before := testInTime.Sub(time.Now())
		d, err := TimestampUntil(test.ts)
		if (err == nil) != test.valid {
			t.Errorf("TimestampUntil(%v) = %q\nwhich is invalid(%v)", test.ts, d, err)
			continue
		}
		if err != nil {
			continue
		}
		after := testInTime.Sub(time.Now())
		if d > before || after > d {
			t.Errorf("between %v and %v\nTimestampUntil(%v) = %q", after, before, test.ts, d)
		}
	}
}

func TestTimestampAfter(t *testing.T) {
	for _, test := range []struct {
		ts   *tspb.Timestamp
		uts  *tspb.Timestamp
		want bool
	}{
		{&tspb.Timestamp{0, 0}, &tspb.Timestamp{0, 0}, false},
		{&tspb.Timestamp{maxValidSeconds - 1, 0}, &tspb.Timestamp{0, 0}, true},
		{&tspb.Timestamp{0, 1e9 - 1}, &tspb.Timestamp{0, 0}, true},
		{&tspb.Timestamp{minValidSeconds, 0}, &tspb.Timestamp{minValidSeconds, 1e9 - 1}, false},
		{&tspb.Timestamp{minValidSeconds, 1e9 - 1}, &tspb.Timestamp{minValidSeconds, 0}, true},
		{&tspb.Timestamp{maxValidSeconds, 0}, &tspb.Timestamp{maxValidSeconds, 1e9 - 1}, false},
		{&tspb.Timestamp{maxValidSeconds, 1e9 - 1}, &tspb.Timestamp{maxValidSeconds, 0}, true},
	} {
		tsInTime, _ := Timestamp(test.ts)
		utsInTime, _ := Timestamp(test.uts)
		got := TimestampAfter(test.ts, test.uts)
		if got != test.want {
			t.Errorf("TimestampAfter(%v, %v) = %v, want %v", test.ts, test.uts, got, test.want)
		}
		gotFromTimes := tsInTime.After(utsInTime)
		if got != gotFromTimes {
			t.Errorf("TimestampAfter(%v, %v) = %v does not match with time.After(%v, %v) = %v", test.ts, test.uts, got, tsInTime, utsInTime, gotFromTimes)
		}
	}
}

func TestTimestampBefore(t *testing.T) {
	for _, test := range []struct {
		ts   *tspb.Timestamp
		uts  *tspb.Timestamp
		want bool
	}{
		{&tspb.Timestamp{0, 0}, &tspb.Timestamp{0, 0}, false},
		{&tspb.Timestamp{maxValidSeconds - 1, 0}, &tspb.Timestamp{0, 0}, false},
		{&tspb.Timestamp{0, 1e9 - 1}, &tspb.Timestamp{0, 0}, false},
		{&tspb.Timestamp{minValidSeconds, 0}, &tspb.Timestamp{minValidSeconds, 1e9 - 1}, true},
		{&tspb.Timestamp{minValidSeconds, 1e9 - 1}, &tspb.Timestamp{minValidSeconds, 0}, false},
		{&tspb.Timestamp{maxValidSeconds, 0}, &tspb.Timestamp{maxValidSeconds, 1e9 - 1}, true},
		{&tspb.Timestamp{maxValidSeconds, 1e9 - 1}, &tspb.Timestamp{maxValidSeconds, 0}, false},
	} {
		tsInTime, _ := Timestamp(test.ts)
		utsInTime, _ := Timestamp(test.uts)
		got := TimestampBefore(test.ts, test.uts)
		if got != test.want {
			t.Errorf("TimestampBefore(%v, %v) = %v, want %v", test.ts, test.uts, got, test.want)
		}
		gotFromTimes := tsInTime.Before(utsInTime)
		if got != gotFromTimes {
			t.Errorf("TimestampBefore(%v, %v) = %v does not match with time.Before(%v, %v) = %v", test.ts, test.uts, got, tsInTime, utsInTime, gotFromTimes)
		}
	}
}

func TestTimestampEqual(t *testing.T) {
	for _, test := range []struct {
		ts   *tspb.Timestamp
		uts  *tspb.Timestamp
		want bool
	}{
		{&tspb.Timestamp{0, 0}, &tspb.Timestamp{0, 0}, true},
		{&tspb.Timestamp{minValidSeconds, 0}, &tspb.Timestamp{minValidSeconds, 0}, true},
		{&tspb.Timestamp{minValidSeconds, 0}, &tspb.Timestamp{minValidSeconds, 1e9 - 1}, false},
		{&tspb.Timestamp{minValidSeconds, 1e9 - 1}, &tspb.Timestamp{minValidSeconds, 1e9 - 1}, true},
	} {
		tsInTime, _ := Timestamp(test.ts)
		utsInTime, _ := Timestamp(test.uts)
		got := TimestampEqual(test.ts, test.uts)
		if got != test.want {
			t.Errorf("TimestampEqual(%v, %v) = %v, want %v", test.ts, test.uts, got, test.want)
		}
		gotFromTimes := tsInTime.Equal(utsInTime)
		if got != gotFromTimes {
			t.Errorf("TimestampEqual(%v, %v) = %v does not match with time.Equal(%v, %v) = %v", test.ts, test.uts, got, tsInTime, utsInTime, gotFromTimes)
		}
	}

}
