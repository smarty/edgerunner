package edgerunner

import (
	"context"
	"errors"
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

func TestFixture(t *testing.T) {
	gunit.Run(new(Fixture), t, gunit.Options.AllSequential())
}

type Fixture struct {
	*gunit.Fixture
	ctx   context.Context
	tasks []*TaskForTests
}

func delay() time.Duration {
	return time.Millisecond * 100
}
func delayedClose(d time.Duration, closer io.Closer) {
	time.Sleep(d)
	_ = closer.Close()
}
func (this *Fixture) taskFactory(id int, ready func(bool)) Task {
	if id > len(this.tasks) {
		return nil
	}
	task := this.tasks[id]
	task.identify(id, ready)
	return task
}
func (this *Fixture) NewRunner(options ...option) Runner {
	return New(append([]option{
		Options.Context(this.ctx),
		Options.TaskVersion("0"),
		Options.TaskName(this.Name()),
		Options.TaskFactory(this.taskFactory),
		Options.Logger(NewTestLogger(this.T(), "EDGE")),
		Options.WatchReloadSignals(syscall.SIGUSR1),
		Options.WatchTerminateSignals(syscall.SIGUSR2),
	}, options...)...)
}
func (this *Fixture) Listen(runner Runner, tasks ...*TaskForTests) {
	this.tasks = tasks
	runner.Listen()
}
func (this *Fixture) Setup() {
	this.ctx = context.WithValue(context.Background(), "name", this.Name())
}
func (this *Fixture) TestTask_NoTaskFactory_Panic() {
	this.So(func() { New() }, should.Panic)
}
func (this *Fixture) TestTaskFactoryReturnsNilTask() {
	runner := this.NewRunner(Options.TaskFactory(func(id int, ready func(bool)) Task { return nil }))
	go delayedClose(delay(), runner)
	this.Listen(runner)
	this.So(runner.(*defaultRunner).id, should.Equal, 0)
}
func (this *Fixture) TestTask_InitializationError() {
	task := NewTaskForTests(NewTestLogger(this.T(), "TASK"), omitReadiness())
	task.initErr = errors.New("BOINK")

	runner := this.NewRunner()
	go delayedClose(delay(), runner)
	this.Listen(runner, task)

	time.Sleep(delay())
	this.So(task.id, should.Equal, 0)
	this.So(task.initialized.Load(), should.Equal, 1)
	this.So(task.listened.Load(), should.Equal, 0)
	this.So(task.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestTask_Initialized_Listened_Closed() {
	task := NewTaskForTests(NewTestLogger(this.T(), "TASK"), prepared())

	runner := this.NewRunner()
	go delayedClose(delay(), runner)
	this.Listen(runner, task)

	this.So(task.id, should.Equal, 0)
	this.So(task.initialized.Load(), should.Equal, 1)
	this.So(task.listened.Load(), should.Equal, 1)
	this.So(task.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestReload() {
	task1 := NewTaskForTests(NewTestLogger(this.T(), "TASK-1"), prepared())
	task2 := NewTaskForTests(NewTestLogger(this.T(), "TASK-2"), omitReadiness())

	runner := this.NewRunner()

	go func() {
		time.Sleep(delay())
		runner.Reload()
		delayedClose(delay(), runner)
	}()
	this.Listen(runner, task1, task2)

	this.So(task1.id, should.Equal, 0)
	this.So(task1.initialized.Load(), should.Equal, 1)
	this.So(task1.listened.Load(), should.Equal, 1)
	this.So(task1.closed.Load(), should.Equal, 1)

	this.So(task2.id, should.Equal, 1)
	this.So(task2.initialized.Load(), should.Equal, 1)
	this.So(task2.listened.Load(), should.Equal, 1)
	this.So(task2.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestSubsequentTaskFailsReadinessCheck_ClosedImmediately_PreviousTaskContinues() {
	task1 := NewTaskForTests(NewTestLogger(this.T(), "TASK-1"), prepared())
	task2 := NewTaskForTests(NewTestLogger(this.T(), "TASK-2"), unprepared())
	task1.readiness = prepared()
	task2.readiness = unprepared()

	runner := this.NewRunner()
	wait := prepareWaiter(load(func() {
		wait := prepareWaiter(load(func() {
			this.Listen(runner, task1, task2)
		}))
		time.Sleep(delay())
		runner.Reload()
		time.Sleep(delay())

		this.So(task1.closed.Load(), should.Equal, 0)

		this.So(task2.initialized.Load(), should.Equal, 1)
		this.So(task2.listened.Load(), should.Equal, 1)
		this.So(task2.closed.Load(), should.Equal, 1)
		wait()
	}))

	delayedClose(delay()*5, runner)
	wait()

	this.So(task2.initialized.Load(), should.Equal, 1)
	this.So(task2.listened.Load(), should.Equal, 1)
	this.So(task2.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestTerminate() {
	task := NewTaskForTests(NewTestLogger(this.T(), "TASK"), prepared())

	runner := this.NewRunner()
	go func() {
		time.Sleep(delay())
		runner.(*defaultRunner).terminationSignals <- syscall.Signal(0)
		time.Sleep(delay())
	}()
	this.Listen(runner, task)

	this.So(task.id, should.Equal, 0)
	this.So(task.initialized.Load(), should.Equal, 1)
	this.So(task.listened.Load(), should.Equal, 1)
	this.So(task.closed.Load(), should.Equal, 1)
}
func (this *Fixture) TestReadinessTimeoutReached_TaskAborted() {
	task1 := NewTaskForTests(NewTestLogger(this.T(), "TASK-1"), prepared())
	task2 := NewTaskForTests(NewTestLogger(this.T(), "TASK-2"), omitReadiness())

	runner := this.NewRunner(Options.ReadinessTimeout(delay() / 2))
	wait := prepareWaiter(load(func() {
		wait := prepareWaiter(load(func() {
			this.Listen(runner, task1, task2)
		}))
		runner.Reload()
		time.Sleep(delay())
		this.So(task1.closed.Load(), should.Equal, 0)

		this.So(task2.initialized.Load(), should.Equal, 1)
		this.So(task2.listened.Load(), should.Equal, 1)
		this.So(task2.closed.Load(), should.Equal, 1)
		wait()
	}))

	delayedClose(delay()*5, runner)
	wait()

	this.So(task2.initialized.Load(), should.Equal, 1)
	this.So(task2.listened.Load(), should.Equal, 1)
	this.So(task2.closed.Load(), should.Equal, 1)
}
