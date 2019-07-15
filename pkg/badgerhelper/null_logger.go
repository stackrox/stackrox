package badgerhelper

type nullLogger struct{}

func (nullLogger) Infof(string, ...interface{})    {}
func (nullLogger) Warningf(string, ...interface{}) {}
func (nullLogger) Errorf(string, ...interface{})   {}
