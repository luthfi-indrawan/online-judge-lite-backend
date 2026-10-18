package controller

import (
	"net/http"
	"online-judge-lite-backend/internal/model"
	"online-judge-lite-backend/internal/response"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type submissionInput struct {
	ProblemID int    `json:"problem_id" binding:"required,gt=0"`
	Language  string `json:"language" binding:"required,oneof=c cpp"`
	Code      string `json:"code" binding:"required"`
}

func (ctl *Controller) CreateSubmission(c *gin.Context) {
	var in submissionInput
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Code) == "" || len(in.Code) > 200000 {
		response.Fail(c, 400, "VALIDATION_ERROR", "problem_id, language c/cpp, dan code wajib valid (maksimal 200 KB)", nil)
		return
	}
	var start, end string
	err := ctl.DB.QueryRow(c, `SELECT g.start_time::text,g.end_time::text FROM problems p JOIN problem_groups g ON g.id=p.group_id WHERE p.id=$1`, in.ProblemID).Scan(&start, &end)
	if err != nil {
		response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Soal tidak ditemukan", nil)
		return
	}
	var open bool
	err = ctl.DB.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM problems p JOIN problem_groups g ON g.id=p.group_id WHERE p.id=$1 AND CURRENT_TIMESTAMP BETWEEN g.start_time AND g.end_time)`, in.ProblemID).Scan(&open)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal memeriksa jadwal kontes", nil)
		return
	}
	if !open {
		response.Fail(c, 403, "CONTEST_CLOSED", "Submission hanya dapat dilakukan selama kontes berlangsung", nil)
		return
	}
	var s model.Submission
	err = ctl.DB.QueryRow(c, `INSERT INTO submissions(user_id,problem_id,language,code,verdict) VALUES($1,$2,$3::programming_language,$4,'PENDING') RETURNING id,user_id,problem_id,language::text,verdict::text,execution_time_ms,submitted_at`, uid(c), in.ProblemID, in.Language, in.Code).Scan(&s.ID, &s.UserID, &s.ProblemID, &s.Language, &s.Verdict, &s.ExecutionTimeMS, &s.SubmittedAt)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal menyimpan submission", nil)
		return
	}
	select {
	case ctl.Queue <- s.ID:
		response.OK(c, 202, "Submission diterima dan sedang diproses dalam antrean", gin.H{"submission_id": s.ID, "status": s.Verdict, "submitted_at": s.SubmittedAt})
	default:
		_, _ = ctl.DB.Exec(c, `UPDATE submissions SET verdict='RTE',compile_output='Submission queue is full; please retry.' WHERE id=$1`, s.ID)
		response.Fail(c, http.StatusServiceUnavailable, "QUEUE_FULL", "Antrean submission sedang penuh, silakan coba lagi", nil)
	}
}
func (ctl *Controller) GetSubmission(c *gin.Context) {
	var s model.Submission
	err := ctl.DB.QueryRow(c, `SELECT id,user_id,problem_id,language::text,code,verdict::text,execution_time_ms,COALESCE(compile_output,''),submitted_at FROM submissions WHERE id=$1`, c.Param("id")).Scan(&s.ID, &s.UserID, &s.ProblemID, &s.Language, &s.Code, &s.Verdict, &s.ExecutionTimeMS, &s.CompileOutput, &s.SubmittedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Submission tidak ditemukan", nil)
		} else {
			response.Fail(c, 400, "VALIDATION_ERROR", "ID submission tidak valid", nil)
		}
		return
	}
	if !isAdmin(c) && s.UserID != uid(c) {
		response.Fail(c, 403, "FORBIDDEN_ACCESS", "Anda hanya dapat melihat submission milik sendiri", nil)
		return
	}
	response.OK(c, 200, "Detail submission berhasil diambil", s)
}
func (ctl *Controller) ListMySubmissions(c *gin.Context) {
	rows, err := ctl.DB.Query(c, `SELECT id,user_id,problem_id,language::text,verdict::text,execution_time_ms,COALESCE(compile_output,''),submitted_at FROM submissions WHERE user_id=$1 ORDER BY submitted_at DESC LIMIT 100`, uid(c))
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal mengambil riwayat submission", nil)
		return
	}
	defer rows.Close()
	items := []gin.H{}
	for rows.Next() {
		var s model.Submission
		if rows.Scan(&s.ID, &s.UserID, &s.ProblemID, &s.Language, &s.Verdict, &s.ExecutionTimeMS, &s.CompileOutput, &s.SubmittedAt) == nil {
			items = append(items, gin.H{"id": s.ID, "user_id": s.UserID, "problem_id": s.ProblemID, "language": s.Language, "verdict": s.Verdict, "execution_time_ms": s.ExecutionTimeMS, "compile_output": s.CompileOutput, "submitted_at": s.SubmittedAt})
		}
	}
	response.OK(c, 200, "Riwayat submission berhasil diambil", items)
}
