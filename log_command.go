package oslog_collector

import "os/exec"

var (
	_ LogCommandRunnerGenerator = NewLogCommandRunner
	_ LogCommandRunner          = &logCommandRunner{}
)

type LogCommandRunnerGenerator func(args []string) LogCommandRunner

type LogCommandRunner interface {
	RunLogCommand() ([]byte, error)
}

type logCommandBuilder struct {
	command []string
}

type logCommandRunner struct {
	cmd *exec.Cmd
}

func NewLogCommandRunner(args []string) LogCommandRunner {
	return &logCommandRunner{
		cmd: exec.Command(args[0], args[1:]...),
	}
}

func (r *logCommandRunner) RunLogCommand() ([]byte, error) {
	return r.cmd.CombinedOutput()
}

func NewLogCommandBuilder() *logCommandBuilder {
	return &logCommandBuilder{
		command: []string{"log", "show"},
	}
}

func (b *logCommandBuilder) WithPredicate(predicate string) *logCommandBuilder {
	b.command = append(b.command, "--predicate", predicate)
	return b
}

func (b *logCommandBuilder) WithStartTime(startTime string) *logCommandBuilder {
	b.command = append(b.command, "--start", startTime)
	return b
}

func (b *logCommandBuilder) WithEndTime(endTime string) *logCommandBuilder {
	b.command = append(b.command, "--end", endTime)
	return b
}

func (b *logCommandBuilder) WithStyle(style string) *logCommandBuilder {
	b.command = append(b.command, "--style", style)
	return b
}

func (b *logCommandBuilder) WithInfoLevel(enable bool) *logCommandBuilder {
	if enable {
		b.command = append(b.command, "--info")
	}
	return b
}

func (b *logCommandBuilder) Build() []string {
	return b.command
}
