// Code generated by "stringer -type=ScanResult"; DO NOT EDIT.

package enricher

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ScanNotDone-0]
	_ = x[ScanTriggered-1]
	_ = x[ScanSucceeded-2]
}

const _ScanResult_name = "ScanNotDoneScanTriggeredScanSucceeded"

var _ScanResult_index = [...]uint8{0, 11, 24, 37}

func (i ScanResult) String() string {
	if i < 0 || i >= ScanResult(len(_ScanResult_index)-1) {
		return "ScanResult(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ScanResult_name[_ScanResult_index[i]:_ScanResult_index[i+1]]
}
