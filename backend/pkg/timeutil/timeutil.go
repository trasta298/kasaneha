package timeutil

import (
	"time"
)

// JST is the Japan Standard Time location
var JST *time.Location

func init() {
	var err error
	JST, err = time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// Fallback to UTC+9 if loading location fails
		JST = time.FixedZone("JST", 9*60*60)
	}
}

// NowJST returns the current time in JST
func NowJST() time.Time {
	return time.Now().In(JST)
}

// InJST converts a time to JST
func InJST(t time.Time) time.Time {
	return t.In(JST)
}

// ParseDateInJST parses a date string in JST
func ParseDateInJST(layout, value string) (time.Time, error) {
	t, err := time.ParseInLocation(layout, value, JST)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// FormatJST formats a time in JST
func FormatJST(t time.Time, layout string) string {
	return t.In(JST).Format(layout)
}

// TodayJST returns today's date in JST formatted as "2006-01-02"
func TodayJST() string {
	return NowJST().Format("2006-01-02")
}

// BeginningOfDayJST returns the beginning of the day (00:00:00) for the given time in JST
func BeginningOfDayJST(t time.Time) time.Time {
	jst := t.In(JST)
	return time.Date(jst.Year(), jst.Month(), jst.Day(), 0, 0, 0, 0, JST)
}

// EndOfDayJST returns the end of the day (23:59:59.999999999) for the given time in JST
func EndOfDayJST(t time.Time) time.Time {
	jst := t.In(JST)
	return time.Date(jst.Year(), jst.Month(), jst.Day(), 23, 59, 59, 999999999, JST)
}
