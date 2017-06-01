package timestamp

import (
	"testing"
	"time"
)

func TestTimestampsConversions(t *testing.T) {

	times := []time.Time{
		time.Time{},
		time.Date(2014, time.January, 1, 1, 1, 1, 1, time.UTC),
		time.Date(2015, time.February, 2, 1, 1, 1, 1, time.UTC),
		time.Date(2016, time.March, 1, 2, 1, 1, 1, time.UTC),
		time.Date(2017, time.April, 1, 1, 2, 1, 1, time.UTC),
		time.Date(2018, time.May, 1, 1, 1, 2, 1, time.UTC),
		time.Date(2019, time.June, 1, 1, 1, 1, 2, time.UTC),
		time.Date(2020, time.July, 2, 2, 2, 2, 2, time.UTC),
		time.Date(2021, time.August, 2, 2, 2, 2, 2, time.UTC),
		time.Date(2021, time.August, 2, 2, 2, 2, 9999999999, time.UTC),
	}

	for _, tc := range times {
		ts := NewTimestamp(tc)
		tsTime := ts.Time()

		// equality can't be true but difference
		// gives us approximate correctness
		if tsTime.Sub(tc) != 0 {
			t.Errorf("NewDuration(%s).Duration() != %[1]s. Duration: %[2]v. Diff: %s", tc, ts, tsTime.Sub(tc))
		}
	}

}
