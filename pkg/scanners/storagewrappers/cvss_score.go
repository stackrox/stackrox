package storagewrappers

import (
	"github.com/stackrox/rox/generated/storage"
)

type CVSSScoreWriter interface {
	AsCVSSScore() *storage.CVSSScore
	SetSource(source storage.Source)
	SetURL(url string)
	CVSSV2ScoreWrapper() CVSSV2Writer
	CVSSV3ScoreWrapper() CVSSV3Writer
}

type CVSSScoreWrapper struct {
	*storage.CVSSScore
}

var _ CVSSScoreWriter = (*CVSSScoreWrapper)(nil)

func (w *CVSSScoreWrapper) AsCVSSScore() *storage.CVSSScore {
	if w == nil {
		return nil
	}
	return w.CVSSScore
}

func (w *CVSSScoreWrapper) SetSource(source storage.Source) {
	if w == nil || w.CVSSScore == nil {
		return
	}
	w.Source = source
}

func (w *CVSSScoreWrapper) SetURL(url string) {
	if w == nil || w.CVSSScore == nil {
		return
	}
	w.Url = url
}

func (w *CVSSScoreWrapper) CVSSV2ScoreWrapper() CVSSV2Writer {
	if w == nil || w.CVSSScore == nil {
		return nil
	}
	if w.CvssScore != nil {
		// CVSS V3 takes precedence over V2
		v2, ok := w.CvssScore.(*storage.CVSSScore_Cvssv2)
		if ok {
			return &CVSSV2Wrapper{CVSSV2: v2.Cvssv2}
		}
		return nil
	}
	cvssV2 := &storage.CVSSV2{}
	w.CvssScore = &storage.CVSSScore_Cvssv2{Cvssv2: cvssV2}
	return &CVSSV2Wrapper{CVSSV2: cvssV2}
}

func (w *CVSSScoreWrapper) CVSSV3ScoreWrapper() CVSSV3Writer {
	if w == nil || w.CVSSScore == nil {
		return nil
	}
	if w.CvssScore != nil {
		v3, ok := w.CvssScore.(*storage.CVSSScore_Cvssv3)
		if ok {
			return &CVSSV3Wrapper{CVSSV3: v3.Cvssv3}
		}
		// fallthrough:  CVSS V3 takes precedence over V2
	}
	w.CvssScore = &storage.CVSSScore_Cvssv3{Cvssv3: &storage.CVSSV3{}}
	return &CVSSV3Wrapper{CVSSV3: w.GetCvssv3()}
}
