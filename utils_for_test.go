package edgerunner

import (
	"fmt"

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
	this.Log(fmt.Sprintf(this.prefix+" "+format, args...))
}
