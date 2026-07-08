package datastore

import "time"

// CVETimestampsView holds the minimal fields needed for V1 CVE pruning timestamp comparison.
type CVETimestampsView struct {
	ID                    *string    `db:"cve_id"`
	ImageID               *string    `db:"cve_image_id"`
	ImageIDV2             *string    `db:"cve_image_id_v2"`
	CVE                   *string    `db:"cve"`
	IsFixable             *bool      `db:"fixable"`
	CreatedAt             *time.Time `db:"cve_created_time"`
	FirstImageOccurrence  *time.Time `db:"first_image_occurrence_timestamp"`
	FixAvailableTimestamp *time.Time `db:"cve_fix_available_timestamp"`
}

// GetID returns the CVE row ID, or empty string if nil.
func (v *CVETimestampsView) GetID() string {
	if v.ID == nil {
		return ""
	}
	return *v.ID
}

// GetImageID returns the V1 image ID (digest), or empty string if nil.
func (v *CVETimestampsView) GetImageID() string {
	if v.ImageID == nil {
		return ""
	}
	return *v.ImageID
}

// GetImageIDV2 returns the V2 image ID, or empty string if nil.
func (v *CVETimestampsView) GetImageIDV2() string {
	if v.ImageIDV2 == nil {
		return ""
	}
	return *v.ImageIDV2
}

// GetCVE returns the CVE name, or empty string if nil.
func (v *CVETimestampsView) GetCVE() string {
	if v.CVE == nil {
		return ""
	}
	return *v.CVE
}

// GetIsFixable returns whether the CVE is fixable, or false if nil.
func (v *CVETimestampsView) GetIsFixable() bool {
	if v.IsFixable == nil {
		return false
	}
	return *v.IsFixable
}

// GetCreatedAt returns the created_at timestamp, or nil if not set.
func (v *CVETimestampsView) GetCreatedAt() *time.Time {
	return v.CreatedAt
}

// GetFirstImageOccurrence returns the first_image_occurrence timestamp, or nil if not set.
func (v *CVETimestampsView) GetFirstImageOccurrence() *time.Time {
	return v.FirstImageOccurrence
}

// GetFixAvailableTimestamp returns the fix_available_timestamp, or nil if not set.
func (v *CVETimestampsView) GetFixAvailableTimestamp() *time.Time {
	return v.FixAvailableTimestamp
}
