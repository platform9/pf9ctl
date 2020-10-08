package log

// Formatted Info level log
func Infof(template string, args ...interface{}) {
	s.Infof(template, args)
}

// Formatted Debug level log
func Debugf(template string, args ...interface{}) {
	s.Debugf(template, args)
}

// Formatted Panic level log
func Panicf(template string, args ...interface{}) {
	s.Panicf(template, args)
}

// Formatted Error level log
func Errorf(template string, args ...interface{}) {
	s.Errorf(template, args)
}

// Formatted Fatal level log
func Fatalf(template string, args ...interface{}) {
	s.Fatalf(template, args)
}
