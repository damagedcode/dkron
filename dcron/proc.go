package dcron

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/armon/circbuf"
	"github.com/hashicorp/serf/serf"
)

const (
	windows = "windows"

	// maxBufSize limits how much data we collect from a handler.
	// This is to prevent Serf's memory from growing to an enormous
	// amount due to a faulty handler.
	maxBufSize = 8 * 1024
)

// spawn command that specified as proc.
func spawnProc(proc string) (*exec.Cmd, error) {
	cs := []string{"/bin/bash", "-c", proc}
	cmd := exec.Command(cs[0], cs[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ())

	log.Printf("Starting %s\n", proc)
	err := cmd.Start()
	if err != nil {
		log.Errorf("Failed to start %s: %s\n", proc, err)
		return nil, err
	}
	return cmd, nil
}

// invokeJob will execute the given job. Depending on the event.
func invokeJob(jobPayload []byte, event serf.Event) error {
	output, _ := circbuf.NewBuffer(maxBufSize)
	var job Job

	json.Unmarshal(jobPayload, &job)

	// Determine the shell invocation based on OS
	var shell, flag string
	if runtime.GOOS == windows {
		shell = "cmd"
		flag = "/C"
	} else {
		shell = "/bin/sh"
		flag = "-c"
	}

	cmd := exec.Command(shell, flag, job.Command)
	cmd.Stderr = output
	cmd.Stdout = output

	// Start a timer to warn about slow handlers
	slowTimer := time.AfterFunc(2*time.Hour, func() {
		log.Warnf("agent: Script '%s' slow, execution exceeding %v", job.Command, 2*time.Hour)
	})

	if err := cmd.Start(); err != nil {
		return err
	}

	// Warn if buffer is overritten
	if output.TotalWritten() > output.Size() {
		log.Warnf("agent: Script '%s' generated %d bytes of output, truncated to %d", job.Command, output.TotalWritten(), output.Size())
	}

	err := cmd.Wait()
	slowTimer.Stop()
	log.Debugf("agent: Command output: %s", output)
	if err != nil {
		return err
	}

	// If this is a query and we have output, respond
	if query, ok := event.(*serf.Query); ok && output.TotalWritten() > 0 {
		if err := query.Respond(output.Bytes()); err != nil {
			log.Warnf("agent: Failed to respond to query '%s': %s", event.String(), err)
		}
	}

	return nil
}
