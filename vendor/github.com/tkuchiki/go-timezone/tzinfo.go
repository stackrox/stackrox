package timezone

// TzInfo represents information about a particular timezone.
type TzInfo struct {
	longGeneric        string
	longStandard       string
	longDaylight       string
	shortGeneric       string
	shortStandard      string
	shortDaylight      string
	standardOffset     int
	daylightOffset     int
	standardOffsetHHMM string
	daylightOffsetHHMM string
	countryCode        string
	isDeprecated       bool
	linkTo             string
	lastDST            int
}

// LongGeneric returns the generic time name.
// It returns the empty string if there is no generic time name.
func (ti *TzInfo) LongGeneric() string {
	return ti.longGeneric
}

// LongStandard returns the standard time name.
func (ti *TzInfo) LongStandard() string {
	return ti.longStandard
}

// LongDaylight returns the daylight saving time name.
// It returns the empty string if there is no daylight saving time name.
func (ti *TzInfo) LongDaylight() string {
	return ti.longDaylight
}

// ShortGeneric returns the generic timezone abbreviation.
// It returns the empty string if there is no generic timezone abbreviation.
func (ti *TzInfo) ShortGeneric() string {
	return ti.shortGeneric
}

// ShortStandard returns the standard timezone abbreviation.
func (ti *TzInfo) ShortStandard() string {
	return ti.shortStandard
}

// ShortDaylight returns the daylight saving timezone abbreviation.
// It returns the empty string if there is no daylight saving timezone abbreviation.
func (ti *TzInfo) ShortDaylight() string {
	return ti.shortDaylight
}

// StandardOffset returns the standard time offset
func (ti *TzInfo) StandardOffset() int {
	return ti.standardOffset
}

// DaylightOffset returns the daylight saving time offset
// It returns the 0 if there is no daylight saving.
func (ti *TzInfo) DaylightOffset() int {
	return ti.daylightOffset
}

// StandardOffsetHHMM returns the standard time offset in (+/-)hh:mm format.
func (ti *TzInfo) StandardOffsetHHMM() string {
	return ti.standardOffsetHHMM
}

// DaylightOffsetHHMM returns the daylight saving time offset in (+/-)hh:mm format.
// It returns the "+00:00" if there is no daylight saving.
func (ti *TzInfo) DaylightOffsetHHMM() string {
	return ti.daylightOffsetHHMM
}

// CountryCode returns the ISO 3166-1 alpha-2 country code.
// It returns the empty string if there is no CountryCode.
func (ti *TzInfo) CountryCode() string {
	return ti.countryCode
}

// IsDeprecated reports whether the timezone is deprecated.
func (ti *TzInfo) IsDeprecated() bool {
	return ti.isDeprecated
}

// LinkTo returns the source of the alias.
// It returns the empty string if there is no alias.
func (ti *TzInfo) LinkTo() string {
	return ti.linkTo
}

// LastDST returns the last year when there was daylight savings time.
// It returns the 0 if has never observed daylight savings.
func (ti *TzInfo) LastDST() int {
	return ti.lastDST
}

// HasDST reports whether or not it has daylight savings time.
func (ti *TzInfo) HasDST() bool {
	return ti.longDaylight != ""
}

// TzAbbreviationInfo represents timezone abbreviation information about a particular timezone.
type TzAbbreviationInfo struct {
	countryCode string
	isDST       bool
	name        string
	offset      int
	offsetHHMM  string
}

// CountryCode returns the ISO 3166-1 alpha-2 country code.
// It returns the empty string if there is no CountryCode.
func (tai *TzAbbreviationInfo) CountryCode() string {
	return tai.countryCode
}

// IsDST reports whether or not it is daylight savings time.
func (tai *TzAbbreviationInfo) IsDST() bool {
	return tai.isDST
}

// Name returns the time name.
func (tai *TzAbbreviationInfo) Name() string {
	return tai.name
}

// Offset returns the time offset.
func (tai *TzAbbreviationInfo) Offset() int {
	return tai.offset
}

// OffsetHHMM returns the time offset in (+/-)hh:mm format.
func (tai *TzAbbreviationInfo) OffsetHHMM() string {
	return tai.offsetHHMM
}
