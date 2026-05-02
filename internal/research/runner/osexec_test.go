package runner

import (
	"context"
	"io"
	"strings"
	"testing"
)

func TestOSExec_nonZeroExitIncludesStderrTailAndCode(t *testing.T) {
	// Use /bin/sh as a stand-in for claude: it exits non-zero and writes
	// to stderr, exactly the failure mode we want OSExec.cmdReader.Close()
	// to surface. The format string is hardcoded "claude" since claude
	// is the only production bin; the test verifies the diagnostic pieces.
	ex := OSExec{}
	rc, err := ex.Run(context.Background(), Command{
		Bin:  "/bin/sh",
		Args: []string{"-c", "printf 'oops something broke\\n' >&2; exit 7"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, rerr := io.ReadAll(rc); rerr != nil {
		t.Fatalf("ReadAll stdout: %v", rerr)
	}

	cerr := rc.Close()
	if cerr == nil {
		t.Fatalf("Close: want non-nil error from non-zero exit")
	}
	msg := cerr.Error()
	if !strings.Contains(msg, "exited 7") {
		t.Errorf("error = %q; want one containing 'exited 7'", msg)
	}
	if !strings.Contains(msg, "oops something broke") {
		t.Errorf("error = %q; want one containing the stderr tail 'oops something broke'", msg)
	}
}

func TestOSExec_stderrTailIsBounded(t *testing.T) {
	// Flood stderr with much more than the 16KB tail bound, then assert
	// the close error contains a sentinel that lives in the last bytes
	// but does NOT contain a sentinel from the very first bytes.
	const sentinelTail = "TAIL_SENTINEL_KEEP_ME"
	const sentinelHead = "HEAD_SENTINEL_DROP_ME"
	// 32KB of filler beats the 16KB bound by 2x with margin.
	script := "printf '" + sentinelHead + "\\n' >&2; " +
		"head -c 32768 /dev/zero | tr '\\0' x >&2; " +
		"printf '" + sentinelTail + "\\n' >&2; exit 1"

	ex := OSExec{}
	rc, err := ex.Run(context.Background(), Command{
		Bin:  "/bin/sh",
		Args: []string{"-c", script},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, rerr := io.ReadAll(rc); rerr != nil {
		t.Fatalf("ReadAll: %v", rerr)
	}

	cerr := rc.Close()
	if cerr == nil {
		t.Fatalf("Close: want non-nil error")
	}
	msg := cerr.Error()
	if !strings.Contains(msg, sentinelTail) {
		t.Errorf("error = %q; want last bytes of stderr in the message", msg)
	}
	if strings.Contains(msg, sentinelHead) {
		t.Errorf("error = %q; head sentinel must have been evicted from the bounded tail", msg)
	}
}

func TestOSExec_zeroExitReturnsNilCloseError(t *testing.T) {
	ex := OSExec{}
	rc, err := ex.Run(context.Background(), Command{
		Bin:  "/bin/sh",
		Args: []string{"-c", "echo hi; printf 'noise on stderr\\n' >&2; exit 0"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, rerr := io.ReadAll(rc); rerr != nil {
		t.Fatalf("ReadAll: %v", rerr)
	}
	if cerr := rc.Close(); cerr != nil {
		t.Fatalf("Close: want nil on zero exit (stderr noise must not be a failure), got %v", cerr)
	}
}
