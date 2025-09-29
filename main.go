package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	Signal   int    `json:"signal,omitempty"`
	TimeMs   int64  `json:"time_ms,omitempty"`
	MaxRSSKB int64  `json:"max_rss_kb,omitempty"`
}

type JudgeResponse struct {
	CompileOK     bool         `json:"compile_ok"`
	CompileStderr string       `json:"compile_stderr,omitempty"`
	Results       []TestResult `json:"results"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /judge", handleJudge)

	addr := ":8080"
	fmt.Println("Listening on", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}

func handleJudge(w http.ResponseWriter, r *http.Request) {
	var req JudgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, "No test cases provided", http.StatusBadRequest)
		return
	}

	workdir, err := os.MkdirTemp("", "seva-run-*")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(workdir)

	sourcePath := filepath.Join(workdir, "main.cpp")
	if err := os.WriteFile(sourcePath, []byte(req.SourceCPP), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	binPath := filepath.Join(workdir, "prog")

}

func compileCPP(src, out string) (string, error) {
	cmd := exec.Command("g++", "-O2", "-pipe", "-std=gnu++17", src, "-o", out)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.String(), err
}
