package repository

import "time"

// DateRange is inclusive on both calendar days (UTC).
type DateRange struct {
	From time.Time
	To   time.Time
}

func (d DateRange) FromISO() string {
	return d.From.Format("2006-01-02")
}

func (d DateRange) ToISO() string {
	return d.To.Format("2006-01-02")
}

// EndExclusive returns start of the day after To (for SQL created_at < end).
func (d DateRange) EndExclusive() time.Time {
	return d.To.Add(24 * time.Hour)
}
