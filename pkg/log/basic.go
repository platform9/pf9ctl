package log

// Info level log
func Info(args ...interface{}) {
	s.Info(args)
}

// Debug level log
func Debug(args ...interface{}) {
	s.Debug(args)
}

// Panic level log
func Panic(args ...interface{}) {
	s.Panic(args)
}

// Error level log
func Error(args ...interface{}) {
	s.Error(args)
}

// Fatal level log
func Fatal(args ...interface{}) {
	s.Fatal(args)
}
