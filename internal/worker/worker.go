package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"online-judge-lite-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkerPool struct {
	db    *pgxpool.Pool
	cfg   config.Config
	queue chan int
}

type testcase struct {
	input, expected string
}

type problem struct {
	id                       int
	language, code           string
	timeLimit, memoryLimit int
}

type result struct {
	verdict  string
	duration int
	output   string
}

func New(db *pgxpool.Pool, cfg config.Config, queue chan int) *WorkerPool {
	return &WorkerPool{db: db, cfg: cfg, queue: queue}
}

func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.cfg.WorkerCount; i++ {
		go func(workerID int) {
			log.Printf("[Worker #%d] Started", workerID)
			for {
				select {
				case <-ctx.Done():
					log.Printf("[Worker #%d] Stopping...", workerID)
					return
				case id, ok := <-p.queue:
					if !ok {
						return
					}
					p.process(ctx, id)
				}
			}
		}(i + 1)
	}
	go p.requeuePending(ctx)
}

func (p *WorkerPool) requeuePending(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := p.db.Query(ctx, `SELECT id FROM submissions WHERE verdict='PENDING' ORDER BY submitted_at LIMIT 100`)
			if err != nil {
				continue
			}
			ids := []int{}
			for rows.Next() {
				var id int
				if rows.Scan(&id) == nil {
					ids = append(ids, id)
				}
			}
			rows.Close()
			for _, id := range ids {
				select {
				case p.queue <- id:
				default:
				}
			}
		}
	}
}

func (p *WorkerPool) process(ctx context.Context, id int) {
	log.Printf("[Worker] Memproses submission ID: %d", id)

	tag, claimErr := p.db.Exec(ctx, `UPDATE submissions SET verdict='RUNNING' WHERE id=$1 AND verdict='PENDING'`, id)
	if claimErr != nil || tag.RowsAffected() == 0 {
		return
	}

	var pr problem
	err := p.db.QueryRow(ctx, `SELECT p.id,s.language::text,s.code,p.time_limit_ms,p.memory_limit_mb FROM submissions s JOIN problems p ON p.id=s.problem_id WHERE s.id=$1`, id).Scan(&pr.id, &pr.language, &pr.code, &pr.timeLimit, &pr.memoryLimit)
	if err != nil {
		p.finish(ctx, id, "RTE", 0, "Failed to load submission")
		log.Printf("[Worker] Submission ID %d gagal: Gagal memuat submission dari DB", id)
		return
	}

	rows, err := p.db.Query(ctx, `SELECT input_data,expected_output FROM test_cases WHERE problem_id=$1 ORDER BY id`, pr.id)
	if err != nil {
		p.finish(ctx, id, "RTE", 0, "Failed to load test cases")
		log.Printf("[Worker] Submission ID %d gagal: Gagal memuat test cases", id)
		return
	}

	cases := []testcase{}
	for rows.Next() {
		var tc testcase
		if rows.Scan(&tc.input, &tc.expected) == nil {
			cases = append(cases, tc)
		}
	}
	rows.Close()

	if len(cases) == 0 {
		p.finish(ctx, id, "RTE", 0, "No test cases configured for this problem")
		log.Printf("[Worker] Submission ID %d gagal: Tidak ada testcase", id)
		return
	}

	verdict := "AC"
	totalMS := 0
	logs := []string{}

	for i, tc := range cases {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res := p.runCase(ctx, id, pr, tc, i)
		totalMS += res.duration
		if res.output != "" && res.verdict != "AC" {
			logs = append(logs, res.output)
		}
		if res.verdict != "AC" {
			verdict = res.verdict
			break
		}
	}

	p.finish(ctx, id, verdict, totalMS, strings.TrimSpace(strings.Join(logs, "\n")))
	log.Printf("[Worker] Selesai submission ID %d dengan verdict: %s (Durasi: %d ms)", id, verdict, totalMS)
}

func (p *WorkerPool) runCase(ctx context.Context, id int, pr problem, tc testcase, index int) result {
	started := time.Now()

	if err := os.MkdirAll(p.cfg.SandboxWorkdir, 0700); err != nil {
		return result{verdict: "RTE", output: "Unable to create sandbox root"}
	}

	dir, err := os.MkdirTemp(p.cfg.SandboxWorkdir, fmt.Sprintf("submission-%d-", id))
	if err != nil {
		return result{verdict: "RTE", output: "Unable to create sandbox workspace"}
	}
	defer os.RemoveAll(dir)

	srcName := "main.cpp"
	compiler := "g++"
	if pr.language == "c" {
		srcName = "main.c"
		compiler = "gcc"
	}

	if err = os.WriteFile(filepath.Join(dir, srcName), []byte(pr.code), 0600); err != nil {
		return result{verdict: "RTE", output: "Unable to write source code"}
	}
	if err = os.WriteFile(filepath.Join(dir, "input.txt"), []byte(tc.input), 0600); err != nil {
		return result{verdict: "RTE", output: "Unable to write test input"}
	}

	// 1. Kompilasi dengan Timeout
	compileCtx, cancel := context.WithTimeout(ctx, time.Duration(p.cfg.CompileTimeoutSeconds)*time.Second)
	defer cancel()

	absDir, absErr := filepath.Abs(dir)
	if absErr != nil {
		return result{verdict: "RTE", output: "Unable to resolve sandbox path"}
	}

	compileArgs := p.dockerArgs(pr, absDir)
	compileArgs = append(compileArgs, "sh", "-lc", compiler+" -O2 -pipe "+map[bool]string{true: "-std=c11", false: "-std=c++17"}[pr.language == "c"]+" "+srcName+" -o /workspace/program")

	var compileOut bytes.Buffer
	cmd := exec.CommandContext(compileCtx, p.cfg.DockerBin, compileArgs...)
	cmd.Stdout = &compileOut
	cmd.Stderr = &compileOut

	err = cmd.Run()
	if compileCtx.Err() == context.DeadlineExceeded {
		return result{verdict: "CE", duration: int(time.Since(started).Milliseconds()), output: "Compilation exceeded compile timeout"}
	}
	if err != nil {
		return result{verdict: "CE", duration: int(time.Since(started).Milliseconds()), output: limit(compileOut.String(), 12000)}
	}

	// 2. Eksekusi Program dengan Timeout
	seconds := float64(pr.timeLimit) / 1000.0
	limitArg := fmt.Sprintf("%.3fs", seconds)

	runArgs := p.dockerArgs(pr, absDir)
	runArgs = append(runArgs, "sh", "-lc", "timeout -s KILL "+limitArg+" ./program < input.txt > output.txt")

	runCtx, cancelRun := context.WithTimeout(ctx, time.Duration(pr.timeLimit+1500)*time.Millisecond)
	defer cancelRun()

	var runOut bytes.Buffer
	runCmd := exec.CommandContext(runCtx, p.cfg.DockerBin, runArgs...)
	runCmd.Stdout = &runOut
	runCmd.Stderr = &runOut

	runErr := runCmd.Run()
	elapsed := int(time.Since(started).Milliseconds())

	if runCtx.Err() == context.DeadlineExceeded {
		return result{verdict: "TLE", duration: elapsed}
	}
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok && (ee.ExitCode() == 124 || ee.ExitCode() == 137 || ee.ExitCode() == -1) {
			return result{verdict: "TLE", duration: elapsed}
		}
		return result{verdict: "RTE", duration: elapsed, output: limit(runOut.String(), 12000)}
	}

	outBytes, readErr := os.ReadFile(filepath.Join(dir, "output.txt"))
	if readErr != nil {
		return result{verdict: "RTE", duration: elapsed, output: "Program did not produce readable output"}
	}

	println(normalize(string(outBytes))) 
	println(normalize(tc.expected)) 
	if normalize(string(outBytes)) != normalize(tc.expected) {
		return result{verdict: "WA", duration: elapsed, output: fmt.Sprintf("Wrong answer on test case #%d", index+1)}
	}

	return result{verdict: "AC", duration: elapsed}
}

func (p *WorkerPool) dockerArgs(pr problem, dir string) []string {
	mem := pr.memoryLimit
	if mem < 16 {
		mem = 16
	}
	return []string{"run", "--rm", "--network", "none", "--memory", fmt.Sprintf("%dm", mem), "--memory-swap", fmt.Sprintf("%dm", mem), "--cpus", "1", "--pids-limit", "64", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid,size=16m", "-v", dir + ":/workspace:rw", "-w", "/workspace", p.cfg.SandboxImage}
}

func (p *WorkerPool) finish(ctx context.Context, id int, verdict string, ms int, output string) {
	_, _ = p.db.Exec(ctx, `UPDATE submissions SET verdict=$1::submission_verdict,execution_time_ms=$2,compile_output=$3 WHERE id=$4`, verdict, ms, output, id)
}

func normalize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

func limit(s string, n int) string {
	if len(s) > n {
		return s[:n] + "\n...[output truncated]"
	}
	return s
}