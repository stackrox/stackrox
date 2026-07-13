package datastore

import "time"

// CVETimeView holds the minimal fields needed for V1 CVE pruning comparison.
type CVETimeView struct {
	ID                   *string    `db:"cve_id"`
	ImageID              *string    `db:"cve_image_id"`
	ImageIDV2            *string    `db:"cve_image_id_v2"`
	CVE                  *string    `db:"cve"`
	FirstImageOccurrence *time.Time `db:"first_image_occurrence_timestamp"`
}

func (v *CVETimeView) GetID() string {
	if v.ID == nil {
		return ""
	}
	return *v.ID
}

func (v *CVETimeView) GetImageID() string {
	if v.ImageID == nil {
		return ""
	}
	return *v.ImageID
}

func (v *CVETimeView) GetImageIDV2() string {
	if v.ImageIDV2 == nil {
		return ""
	}
	return *v.ImageIDV2
}

func (v *CVETimeView) GetCVE() string {
	if v.CVE == nil {
		return ""
	}
	return *v.CVE
}

func (v *CVETimeView) GetFirstImageOccurrence() *time.Time {
	return v.FirstImageOccurrence
}
