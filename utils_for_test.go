package edgerunner

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/smartystreets/gunit"
)

type TestLogger struct {
	gunit.TestingT
	prefix string
}

func NewTestLogger(t gunit.TestingT, prefix string) *TestLogger {
	return &TestLogger{TestingT: t, prefix: prefix}
}
func (this *TestLogger) Printf(format string, args ...any) {
	_, path, line, _ := runtime.Caller(1)
	file := filepath.Base(path)
	clock := fmt.Sprintf("%-12s", time.Now().Format("15:04:05.999"))
	format = fmt.Sprintf("%s %s %s (%s:%d)", clock, this.prefix, format, file, line)
	this.Log(fmt.Sprintf(format, args...))
}
