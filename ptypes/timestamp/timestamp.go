package timestamp

import "time"

//NewTimestamp returns a Timestamp given a time.Time
//
//There will be data loss
func NewTimestamp(t time.Time) *Timestamp {
	return &Timestamp{
		Seconds: t.Unix(),
		Nanos:   int32(t.Nanosecond()),
	}
}

//Time Transforms timestamp to time.Time
//
//There will be data loss
func (m *Timestamp) Time() time.Time {
	return time.Unix(m.Seconds, int64(m.Nanos))
}
