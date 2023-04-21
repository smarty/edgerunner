package edgerunner

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

func TestFixture(t *testing.T) {
	gunit.Run(new(Fixture), t,
		gunit.Options.LongRunning(),
		gunit.Options.AllSequential(),
	)
}

type Fixture struct {
	*gunit.Fixture
	ctx          context.Context
	taskLog      Logger
	edgeLog      Logger
	testDefaults []option
}

func (this *Fixture) allowForInitialization() { time.Sleep(time.Second) }
func (this *Fixture) allowForFinalization()   { time.Sleep(time.Second) }

func (this *Fixture) NewRunner(options ...option) Runner {
	return New(append(this.testDefaults, options...)...)
}
func (this *Fixture) Setup() {
	this.ctx = context.WithValue(context.Background(), "name", this.Name())
	this.taskLog = NewTestLogger(this.T(), "TASK")
	this.edgeLog = NewTestLogger(this.T(), "EDGE")
	this.testDefaults = []option{
		Options.Context(this.ctx),
		Options.TaskVersion("0"),
		Options.TaskName(this.Name()),
		Options.Logger(this.edgeLog),
		Options.WatchReloadSignals(syscall.SIGUSR1),
		Options.WatchTerminateSignals(syscall.SIGUSR2),
	}
}
func (this *Fixture) TestNilTask_Nop() {
	this.NewRunner().Listen() // essentially a no-op, should not block
}
func (this *Fixture) TestTask_InitializationError() {
	task := NewTestingTask(this.taskLog)
	task.initErr = errors.New("BOINK")

	runner := this.NewRunner(
		Options.TaskFactory(func(id int, ready chan<- bool) Task { return task }),
	)

	go runner.Listen()

	this.allowForInitialization()
	this.So(task.initialized.Load(), should.Equal, 1)
	this.So(task.listened.Load(), should.Equal, 0)
	this.So(task.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestTask_Initialized_Listened_Closed() {
	task := NewTestingTask(this.taskLog)
	runner := this.NewRunner(
		Options.TaskFactory(func(id int, ready chan<- bool) Task { return task }),
	)

	go runner.Listen()

	this.allowForInitialization()
	this.So(task.initialized.Load(), should.Equal, 1)

	_ = runner.Close()
	this.allowForFinalization()
	this.So(task.listened.Load(), should.Equal, 1)
	this.So(task.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestReload_NotYetImplemented() {
	this.So(this.NewRunner().Reload, should.PanicWith, "NOT YET IMPLEMENTED")
}
