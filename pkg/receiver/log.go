package receiver

import (
	"fmt"
	"log"
	"os"
)

type Log interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

type nopLog struct{}

func NopLog() Log { return nopLog{} }

func (nopLog) Debugf(string, ...interface{}) {}
func (nopLog) Infof(string, ...interface{})  {}
func (nopLog) Warnf(string, ...interface{})  {}
func (nopLog) Errorf(string, ...interface{}) {}

type stdLog struct {
	logger *log.Logger
	debug  bool
	info   bool
	warn   bool
	error  bool
}

// Creates default logger if none given
func NewStdLog(level string, logger *log.Logger) (Log, error) {
	s := &stdLog{logger: logger}
	if s.logger == nil {
		s.logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	switch level {
	case "debug":
		s.debug = true
		fallthrough
	case "info":
		s.info = true
		fallthrough
	case "warn":
		s.warn = true
		fallthrough
	case "error":
		s.error = true
		fallthrough
	case "off":
	default:
		return nil, fmt.Errorf("unrecognized level %q", level)
	}
	return s, nil
}

func (s *stdLog) Debugf(f string, v ...interface{}) {
	if s.debug {
		s.logger.Printf("[DEBUG] "+f, v...)
	}
}

func (s *stdLog) Infof(f string, v ...interface{}) {
	if s.info {
		s.logger.Printf(" [INFO] "+f, v...)
	}
}

func (s *stdLog) Warnf(f string, v ...interface{}) {
	if s.warn {
		s.logger.Printf(" [WARN] "+f, v...)
	}
}

func (s *stdLog) Errorf(f string, v ...interface{}) {
	if s.error {
		s.logger.Printf("[ERROR] "+f, v...)
	}
}
