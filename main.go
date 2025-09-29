package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type TestCase struct {
	Stdin          string `json:"stdin"`
	ExpectedStdout string `json:"expected_stdout"`
}

type JudgeRequest struct {
	SourceCPP     string     `json:"source_cpp"`
	TimeLimitMs   int        `json:"time_limit_ms"`
	MemoryLimitMB int        `json:"memory_limit_mb"`
	Tests         []TestCase `json:"test_cases"`
}

type TestResult struct {
	Status   string `json:"status"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Reason   string `json:"reason,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Signal   string `json:"signal,omitempty"`
	TimeMs   int64  `json:"time_ms,omitempty"`
	MaxRSSKB int64  `json:"max_rss_kb,omitempty"`
}

type JudgeResponse struct {
	CompileOK     bool         `json:"compile_ok"`
	CompileStderr string       `json:"compile_stderr,omitempty"`
	Results       []TestResult `json:"results"`
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.POST("/judge", handleJudge)

	addr := ":8080"
	fmt.Println("Listening on", addr)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}

func handleJudge(c *gin.Context) {
	var req JudgeRequest
	if err := c.BindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "bad json: %v", err)
		return
	}

	// Default casess
	if req.TimeLimitMs <= 0 {
		req.TimeLimitMs = 1000
	}
	if req.MemoryLimitMB <= 0 {
		req.MemoryLimitMB = 128
	}
	if len(req.Tests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No test cases provided"})
		return
	}

	workdir, err := os.MkdirTemp("", "seva-run-*")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create workdir: %v", err)
		return
	}
	defer os.RemoveAll(workdir)

	sourcePath := filepath.Join(workdir, "main.cpp")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCPP), 0644); err != nil {
		c.String(http.StatusInternalServerError, "failed to write source code: %v", err)
		return
	}

	binPath := filepath.Join(workdir, "prog")

	// COMPILE STEPS
	compileStderr, err := compileCPP(sourcePath, binPath)
	if err != nil {
		c.JSON(http.StatusOK, JudgeResponse{
			CompileOK:     false,
			CompileStderr: compileStderr,
		})
		return
	}

	resp := JudgeResponse{CompileOK: true}
	stopOn := map[string]bool{"CE": true, "RTE": true, "TLE": true, "MLE": true}

	for _, test := range req.Tests {
		result := runWithLimits(binPath, test.Stdin, req.TimeLimitMs, req.MemoryLimitMB, workdir)

		if result.Status == "OK" || result.Status == "" {
			out := strings.TrimSpace(result.Stdout)
			exp := strings.TrimSpace(test.ExpectedStdout)
			if out == exp {
				result.Status = "AC"
			} else {
				result.Status = "WA"
				if result.Reason == "" {
					result.Reason = fmt.Sprintf("expected %q, got %q", exp, out)
				}
			}
		}

		resp.Results = append(resp.Results, result)
		if stopOn[result.Status] {
			break
		}

	}

	c.JSON(http.StatusOK, resp)
}

func compileCPP(src, out string) (string, error) {
	cmd := exec.Command("g++", "-O2", "-pipe", "-std=gnu++17", src, "-o", out)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.String(), err
}

var reMaxRSS = regexp.MustCompile(`(?i)^Maximum resident set size $begin:math:text$kbytes$end:math:text$:\s*([0-9]+)\s*$`)

func runWithLimits(binary, stdin string, timeLimitMs, memLimitMB int, workdir string) TestResult {

	start := time.Now()

	progStderr := filepath.Join(workdir, "stderr.txt")
	timeOut := filepath.Join(workdir, "timeout.txt")
	memKB := memLimitMB * 1024

	sh := fmt.Sprintf(
		`ulimit -v %d; ulimit -s 8192; /usr/bin/time -v -o %q bash -lc 'exec "%s" 2>%q'`,
		memKB, timeOut, binary, progStderr,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeLimitMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-lc", sh)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdin = strings.NewReader(stdin)

	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf

	if err := cmd.Start(); err != nil {
		return TestResult{
			Status: "RTE",
			Reason: fmt.Sprintf("failed to start: %v", err),
		}
	}

	waitErr := cmd.Wait()
	dur := time.Since(start)

	// TLE
	if ctx.Err() == context.DeadlineExceeded {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return TestResult{
			Status: "TLE",
			TimeMs: dur.Milliseconds(),
			Reason: fmt.Sprintf("exceeded %dms", timeLimitMs),
		}
	}

	stderrBytes, _ := os.ReadFile(progStderr)
	stderr := string(stderrBytes)

	var maxRSSKB int64
	if tb, err := os.ReadFile(timeOut); err == nil {
		for _, ln := range strings.Split(string(tb), "\n") {
			if m := reMaxRSS.FindStringSubmatch(strings.TrimSpace(ln)); len(m) == 2 {
				fmt.Sscan(m[1], &maxRSSKB)
				break
			}
		}
	}

	stdout := stdoutBuf.String()

	if waitErr == nil {
		return TestResult{
			Status:   "OK",
			Stdout:   stdout,
			Stderr:   stderr,
			TimeMs:   dur.Milliseconds(),
			MaxRSSKB: maxRSSKB,
		}
	}

	var exitCode int
	var sig string
	if ee := new(exec.ExitError); errors.As(waitErr, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok {
			exitCode = ws.ExitStatus()
			if ws.Signaled() {
				sig = ws.Signal().String()
			}
		}
	}

	status := "RTE"
	reason := "runtime error"
	if sig != "" {
		reason = "terminated by " + sig
		if sig == "killed" || sig == "SIGKILL" {
			status = "MLE"
			reason = "likely memory limit exceeded (SIGKILL)"
		}
	}
	if exitCode == 137 && sig == "" {
		status = "MLE"
		reason = "likely memory limit exceeded (exit 137)"
	}

	return TestResult{
		Status:   status,
		Stdout:   stdout,
		Stderr:   stderr,
		Reason:   reason,
		ExitCode: exitCode,
		Signal:   sig,
		TimeMs:   dur.Milliseconds(),
		MaxRSSKB: maxRSSKB,
	}
}
