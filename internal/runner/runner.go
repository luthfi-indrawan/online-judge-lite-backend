package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"online-judge-lite-backend/internal/config"
)

const (
	maxOutputBytes = 64 * 1024
	maxSourceBytes = 200 * 1024
	maxInputBytes  = 32 * 1024
	defaultTimeMS  = 2000
	defaultMemory  = 128
)

type Request struct {
	Language string
	Code     string
	Input    string
}

type Result struct {
	Status         string `json:"status"`
	Stdout         string `json:"stdout"`
	Stderr         string `json:"stderr"`
	ExecutionTimeMS int   `json:"execution_time_ms"`
}

type Runner struct {
	cfg config.Config
}

func New(cfg config.Config) *Runner {
	return &Runner{cfg: cfg}
}

func (r *Runner) Run(parent context.Context, req Request) Result {
	started := time.Now()

	if req.Language != "c" && req.Language != "cpp" {
		return Result{Status: "INVALID_LANGUAGE"}
	}
	if strings.TrimSpace(req.Code) == "" || len(req.Code) > maxSourceBytes {
		return Result{Status: "INVALID_SOURCE"}
	}
	if len(req.Input) > maxInputBytes {
		return Result{Status: "INVALID_INPUT"}
	}

	if err := os.MkdirAll(r.cfg.SandboxWorkdir, 0700); err != nil {
		return Result{Status: "INTERNAL_ERROR", Stderr: "Unable to create sandbox directory"}
	}

	dir, err := os.MkdirTemp(r.cfg.SandboxWorkdir, "run-")
	if err != nil {
		return Result{Status: "INTERNAL_ERROR", Stderr: "Unable to create sandbox workspace"}
	}
	defer os.RemoveAll(dir)

	srcName := "main.c"
	compiler := "gcc"
	stdFlag := "-std=c11"

	if req.Language == "cpp" {
		srcName = "main.cpp"
		compiler = "g++"
		stdFlag = "-std=c++17"
	}

	if err := os.WriteFile(filepath.Join(dir, srcName), []byte(req.Code), 0600); err != nil {
		return Result{Status: "INTERNAL_ERROR", Stderr: "Unable to write source code"}
	}
	if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte(req.Input), 0600); err != nil {
		return Result{Status: "INTERNAL_ERROR", Stderr: "Unable to write program input"}
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return Result{Status: "INTERNAL_ERROR", Stderr: "Unable to resolve sandbox path"}
	}

	compileSeconds := r.cfg.CompileTimeoutSeconds
	if compileSeconds < 1 {
		compileSeconds = 10
	}

	compileCtx, cancelCompile := context.WithTimeout(parent, time.Duration(compileSeconds)*time.Second)
	defer cancelCompile()

	compileArgs := r.dockerArgs(absDir)
	compileCommand := fmt.Sprintf(
		"%s -O2 -pipe %s %s -o /workspace/program",
		compiler, stdFlag, srcName,
	)
	compileArgs = append(compileArgs, "sh", "-lc", compileCommand)

	var compileOut limitedBuffer
	compileCmd := exec.CommandContext(compileCtx, r.cfg.DockerBin, compileArgs...)
	compileCmd.Stdout = &compileOut
	compileCmd.Stderr = &compileOut

	if err := compileCmd.Run(); err != nil {
		if compileCtx.Err() == context.DeadlineExceeded {
			return Result{
				Status: "COMPILE_TIMEOUT",
				Stderr: "Compilation exceeded the time limit",
				ExecutionTimeMS: int(time.Since(started).Milliseconds()),
			}
		}
		if parent.Err() != nil {
			return Result{Status: "CANCELLED"}
		}
		return Result{
			Status: "COMPILE_ERROR",
			Stderr: limitString(compileOut.String(), maxOutputBytes),
			ExecutionTimeMS: int(time.Since(started).Milliseconds()),
		}
	}

	const timeLimitMS = defaultTimeMS
	runCtx, cancelRun := context.WithTimeout(
		parent,
		time.Duration(timeLimitMS+1500)*time.Millisecond,
	)
	defer cancelRun()

	runArgs := r.dockerArgs(absDir)
	runArgs = append(runArgs, "sh", "-lc", fmt.Sprintf(
		"timeout -s KILL %.3fs ./program < input.txt",
		float64(timeLimitMS)/1000.0,
	))

	var stdout, stderr limitedBuffer
	runCmd := exec.CommandContext(runCtx, r.cfg.DockerBin, runArgs...)
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr

	runErr := runCmd.Run()
	elapsed := int(time.Since(started).Milliseconds())

	if parent.Err() != nil {
		return Result{Status: "CANCELLED", ExecutionTimeMS: elapsed}
	}
	if runCtx.Err() == context.DeadlineExceeded {
		return Result{Status: "TIME_LIMIT_EXCEEDED", ExecutionTimeMS: elapsed}
	}
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok &&
			(exitErr.ExitCode() == 124 || exitErr.ExitCode() == 137) {
			return Result{
				Status: "TIME_LIMIT_EXCEEDED",
				Stdout: stdout.String(),
				Stderr: stderr.String(),
				ExecutionTimeMS: elapsed,
			}
		}
		return Result{
			Status: "RUNTIME_ERROR",
			Stdout: stdout.String(),
			Stderr: limitString(stderr.String(), maxOutputBytes),
			ExecutionTimeMS: elapsed,
		}
	}

	return Result{
		Status: "SUCCESS",
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		ExecutionTimeMS: elapsed,
	}
}

func (r *Runner) dockerArgs(dir string) []string {
	image := strings.TrimSpace(r.cfg.SandboxImage)
	if image == "" {
		image = "online-judge-sandbox:latest"
	}

	return []string{
		"run", "--rm",
		"--network", "none",
		"--memory", fmt.Sprintf("%dm", defaultMemory),
		"--memory-swap", fmt.Sprintf("%dm", defaultMemory),
		"--cpus", "1",
		"--pids-limit", "64",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--read-only",
		"--tmpfs", "/tmp:rw,noexec,nosuid,size=16m",
		"-v", dir + ":/workspace:rw",
		"-w", "/workspace",
		image,
	}
}

type limitedBuffer struct {
	buf bytes.Buffer
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	remaining := maxOutputBytes - b.buf.Len()
	if remaining <= 0 {
		// Terima byte sisanya agar proses tidak gagal hanya karena output dibatasi.
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string {
	return b.buf.String()
}

func limitString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n...[output truncated]"
}
