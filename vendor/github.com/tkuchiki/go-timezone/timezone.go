// Package timezone provides utility for timezone.
package timezone

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrAmbiguousTzAbbreviations indicates ambiguous timezone abbreviations.
	ErrAmbiguousTzAbbreviations = errors.New("Ambiguous timezone abbreviations")

	errFormatInvalidTz             = "Invalid timezone: %s"
	errFormatInvalidTzAbbreviation = "Invalid timezone abbreviation: %s"
	errFormatInvalidTzName         = "Invalid timezone name: %s %s"
	errFormatDoesNotHaveDST        = "Does not have daylight savings: %s"
)

// Timezone represents a timezone information.
type Timezone struct {
	tzInfos     map[string]*TzInfo
	tzAbbrInfos map[string][]*TzAbbreviationInfo
	timezones   map[string][]string
}

// New creates a new Timezone.
func New() *Timezone {
	return &Timezone{
		tzInfos:     tzInfos,
		tzAbbrInfos: tzAbbrInfos,
		timezones:   timezones,
	}
}

// TzInfos returns the all tzInfos.
func (tz *Timezone) TzInfos() map[string]*TzInfo {
	return tz.tzInfos
}

// TzAbbrInfos returns the all tzAbbrInfos.
func (tz *Timezone) TzAbbrInfos() map[string][]*TzAbbreviationInfo {
	return tz.tzAbbrInfos
}

// Timezones returns the all timezones.
func (tz *Timezone) Timezones() map[string][]string {
	return tz.timezones
}

// GetAllTimezones returns the timezones.
//
// Deprecated: Use Timezones.
func (tz *Timezone) GetAllTimezones() map[string][]string {
	return tz.timezones
}

// GetTzAbbreviationInfo returns the slice of TzAbbreviationInfo with the given timezone abbreviation.
// Returns a ErrAmbiguousTzAbbreviations error if timezone abbreviation has more than one meaning.
func (tz *Timezone) GetTzAbbreviationInfo(abbr string) ([]*TzAbbreviationInfo, error) {
	err := fmt.Errorf(errFormatInvalidTzAbbreviation, abbr)
	if _, ok := tz.tzAbbrInfos[abbr]; !ok {
		return nil, err
	}

	if len(tz.tzAbbrInfos[abbr]) > 1 {
		return tz.tzAbbrInfos[abbr], ErrAmbiguousTzAbbreviations
	}

	return tz.tzAbbrInfos[abbr], nil
}

// GetTzAbbreviationInfoByTZName returns the TzAbbreviationInfo with the given timezone abbreviation and timezone name.
// Even if timezone abbreviation has more than one meanings, it can be identified by tzname.
func (tz *Timezone) GetTzAbbreviationInfoByTZName(abbr, tzname string) (*TzAbbreviationInfo, error) {
	if _, ok := tz.tzAbbrInfos[abbr]; !ok {
		return nil, fmt.Errorf(errFormatInvalidTzAbbreviation, abbr)
	}

	for _, tzi := range tz.tzAbbrInfos[abbr] {
		names := strings.Split(tzi.Name(), "/")
		if len(names) == 1 && names[0] == tzname {
			return tzi, nil
		}

		if len(names) == 2 && (names[0] == tzname || names[1] == tzname) {
			return tzi, nil
		}
	}

	return nil, fmt.Errorf(errFormatInvalidTzName, abbr, tzname)
}

// GetTimezones returns the timezones with the given timezone abbreviation.
func (tz *Timezone) GetTimezones(abbr string) ([]string, error) {
	if _, ok := tz.timezones[abbr]; !ok {
		return []string{}, fmt.Errorf(errFormatInvalidTzAbbreviation, abbr)
	}

	return tz.timezones[abbr], nil
}

// FixedTimezone returns the time.Time with the given timezone set from the time.Location.
func (tz *Timezone) FixedTimezone(t time.Time, timezone string) (time.Time, error) {
	var err error
	var loc *time.Location
	zone, offset := time.Now().In(time.Local).Zone()

	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return time.Time{}, err
		}

		return t.In(loc), err
	}

	loc = time.FixedZone(zone, offset)
	return t.In(loc), err
}

// GetTzInfo returns the TzInfo with the given timezone.
func (tz *Timezone) GetTzInfo(timezone string) (*TzInfo, error) {
	tzInfo, ok := tz.tzInfos[timezone]
	if !ok {
		return nil, fmt.Errorf(errFormatInvalidTz, timezone)
	}

	return tzInfo, nil
}

// GetOffset returns the timezone offset with the given timezone abbreviation.
// If also given dst=true, returns the daylight savings timezone offset.
// Returns a ErrAmbiguousTzAbbreviations error if timezone abbreviation has more than one meaning.
//
// Deprecated: Use GetTzAbbreviationInfo or GetTzAbbreviationInfoByTZName
func (tz *Timezone) GetOffset(abbr string, dst ...bool) (int, error) {
	tzAbbrInfos, ok := tz.tzAbbrInfos[abbr]
	if !ok {
		return 0, fmt.Errorf(errFormatInvalidTzAbbreviation, abbr)
	}

	if len(tz.tzAbbrInfos[abbr]) > 1 {
		return 0, ErrAmbiguousTzAbbreviations
	}

	if len(dst) == 0 || !dst[0] {
		return tzAbbrInfos[0].Offset(), nil
	}

	if dst[0] && !tzAbbrInfos[0].IsDST() {
		return 0, fmt.Errorf(errFormatDoesNotHaveDST, abbr)
	}

	return tzAbbrInfos[0].Offset(), nil
}

// GetTimezoneAbbreviation returns the timezone abbreviation with the given timezone.
// If also given dst=true, returns the daylight savings timezone abbreviation.
func (tz *Timezone) GetTimezoneAbbreviation(timezone string, dst ...bool) (string, error) {
	tzinfo, ok := tz.tzInfos[timezone]
	if !ok {
		return "", fmt.Errorf(errFormatInvalidTz, timezone)
	}

	if len(dst) == 0 || !dst[0] {
		return tzinfo.ShortStandard(), nil
	}

	if dst[0] && !tzinfo.HasDST() {
		return "", fmt.Errorf(errFormatDoesNotHaveDST, timezone)
	}

	return tzinfo.ShortDaylight(), nil
}

// IsDST returns whether a given time is daylight saving time or not.
func (tz *Timezone) IsDST(t time.Time) bool {
	t1 := time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
	t2 := time.Date(t.Year(), time.July, 1, 0, 0, 0, 0, t.Location())

	_, tOffset := t.Zone()
	_, t1Offset := t1.Zone()
	_, t2Offset := t2.Zone()

	var dstOffset int
	if t1Offset > t2Offset {
		dstOffset = t1Offset
	} else if t1Offset < t2Offset {
		dstOffset = t2Offset
	} else {
		return false
	}

	if dstOffset == tOffset {
		return true
	}

	return false
}
