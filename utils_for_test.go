package edgerunner

import (
	"fmt"
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
	clock := fmt.Sprintf("%-12s", time.Now().Format("15:04:05.999"))
	format = fmt.Sprintf("%s %s %s", clock, this.prefix, format)
	this.Log(fmt.Sprintf(format, args...))
}
