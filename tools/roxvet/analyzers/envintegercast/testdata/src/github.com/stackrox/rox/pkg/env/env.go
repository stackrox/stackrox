package env

type IntegerSetting struct{}

func RegisterIntegerSetting(string, int) *IntegerSetting { return &IntegerSetting{} }

func (*IntegerSetting) IntegerSetting() int { return 42 }
