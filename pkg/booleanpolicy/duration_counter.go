package booleanpolicy

import "time"

type durationCounter struct {
	message       string
	duration      time.Duration
	lastStartTime time.Time
	count         int64
}

// NewDurationCounter creates a per duration counter
func NewDurationCounter(d time.Duration, message string) *durationCounter {
	return &durationCounter{
		message:       message,
		duration:      d,
		lastStartTime: time.Now(),
		count:         0,
	}
}

func (d *durationCounter) Add() {
	if time.Since(d.lastStartTime) > d.duration {
		log.Infof("Count of %s since %v is %d", d.message, d.lastStartTime, d.count)
		d.count = 0
		d.lastStartTime = time.Now()
	}
	d.count++
}
