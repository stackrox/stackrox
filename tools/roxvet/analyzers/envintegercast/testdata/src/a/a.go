package a

import "github.com/stackrox/rox/pkg/env"

type Setting struct{}

func (Setting) IntegerSetting() int { return 42 }
func (Setting) Setting() string     { return "42" }

var localSetting Setting
var maxRows = env.RegisterIntegerSetting("ROX_TEST_MAX_ROWS", 100)

func directNarrowingCasts() {
	_ = int32(maxRows.IntegerSetting())  // want `avoid direct int32 conversion of IntegerSetting\(\); validate or bound the setting before narrowing`
	_ = uint32(maxRows.IntegerSetting()) // want `avoid direct uint32 conversion of IntegerSetting\(\); validate or bound the setting before narrowing`
}

func allowedCasts() {
	_ = int(maxRows.IntegerSetting())
	value := maxRows.IntegerSetting()
	_ = int32(value)
	_ = int32(0)
	_ = int32(localSetting.IntegerSetting())
	_ = int32(len(localSetting.Setting()))
}
