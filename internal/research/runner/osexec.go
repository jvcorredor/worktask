package runner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// stderrTailCap bounds how much of the child's stderr we hold for diagnostics.
// claude can spew unboundedly on init failures; we keep the last stderrTailCap
// bytes, enough for one or two error lines.
const stderrTailCap = 16 * 1024

// OSExec spawns the real claude binary via os/exec. Production use only.
type OSExec struct{}

func (OSExec) Run(ctx context.Context, cmd Command) (io.ReadCloser, error) {
	c := exec.CommandContext(ctx, cmd.Bin, cmd.Args...)
	c.Stdin = strings.NewReader(cmd.Stdin)

	stderr := &ringBuffer{cap: stderrTailCap}
	c.Stderr = stderr

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("runner: stdout pipe: %w", err)
	}
	if err := c.Start(); err != nil {
		return nil, fmt.Errorf("runner: start %s: %w", cmd.Bin, err)
	}
	return &cmdReader{ReadCloser: stdout, cmd: c, stderr: stderr}, nil
}

type cmdReader struct {
	io.ReadCloser
	cmd    *exec.Cmd
	stderr *ringBuffer
}

func (r *cmdReader) Close() error {
	cerr := r.ReadCloser.Close()
	werr := r.cmd.Wait()
	if werr != nil {
		exitCode := -1
		if exitErr, ok := werr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		tail := strings.TrimSpace(r.stderr.String())
		if tail != "" {
			return fmt.Errorf("runner: claude exited %d: %s", exitCode, tail)
		}
		return fmt.Errorf("runner: claude exited %d: %w", exitCode, werr)
	}
	return cerr
}

// ringBuffer keeps the last cap bytes written to it. Older bytes are
// discarded as new ones arrive. It implements io.Writer so it can be
// plugged into exec.Cmd.Stderr directly; os/exec writes from a goroutine
// alongside Wait, so callers must only read after Wait returns.
type ringBuffer struct {
	buf []byte
	cap int
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if n >= r.cap {
		r.buf = append(r.buf[:0], p[n-r.cap:]...)
		return n, nil
	}
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.cap {
		r.buf = append(r.buf[:0], r.buf[len(r.buf)-r.cap:]...)
	}
	return n, nil
}

func (r *ringBuffer) String() string { return string(r.buf) }
