package log

import "log"

var DebugEnabled bool = false

func Printf(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}

func Debugf(fmt string, args ...interface{}) {
	if DebugEnabled {
		log.Printf(fmt, args...)
	}
}

func Fatalf(fmt string, args ...interface{}) {
	log.Fatalf(fmt, args...)
}
