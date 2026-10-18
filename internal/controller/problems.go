package controller

import (
	"net/url"
	"online-judge-lite-backend/internal/model"
	"online-judge-lite-backend/internal/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type problemInput struct {
	GroupID       int    `json:"group_id" binding:"required,gt=0"`
	Title         string `json:"title" binding:"required"`
	PDFURL        string `json:"pdf_url" binding:"required,url"`
	TimeLimitMS   int    `json:"time_limit_ms"`
	MemoryLimitMB int    `json:"memory_limit_mb"`
}
type testCaseInput struct {
	InputData      string `json:"input_data"`
	ExpectedOutput string `json:"expected_output"`
	IsSample       bool   `json:"is_sample"`
}

func validPDFURL(raw string) bool {
	u, e := url.ParseRequestURI(raw)
	return e == nil && u.Scheme == "https" && u.Host != "" && strings.Contains(strings.ToLower(u.Host), "supabase.co") && strings.Contains(u.Path, "/storage/v1/object/")
}
func (ctl *Controller) CreateProblem(c *gin.Context) {
	var in problemInput
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Title) == "" || !validPDFURL(in.PDFURL) {
		response.Fail(c, 400, "VALIDATION_ERROR", "group_id, title, dan URL PDF Supabase yang valid wajib diisi", nil)
		return
	}
	if in.TimeLimitMS <= 0 {
		in.TimeLimitMS = 1000
	}
	if in.MemoryLimitMB <= 0 {
		in.MemoryLimitMB = 128
	}
	var p model.Problem
	err := ctl.DB.QueryRow(c, `INSERT INTO problems(group_id,title,pdf_url,time_limit_ms,memory_limit_mb) VALUES($1,$2,$3,$4,$5) RETURNING id,group_id,title,pdf_url,time_limit_ms,memory_limit_mb,created_at`, in.GroupID, in.Title, in.PDFURL, in.TimeLimitMS, in.MemoryLimitMB).Scan(&p.ID, &p.GroupID, &p.Title, &p.PDFURL, &p.TimeLimitMS, &p.MemoryLimitMB, &p.CreatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Grup soal tidak ditemukan", nil)
		} else {
			response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal membuat soal", nil)
		}
		return
	}
	response.OK(c, 201, "Soal berhasil dibuat", p)
}
func (ctl *Controller) GetProblem(c *gin.Context) {
	var p model.Problem
	var start, end time.Time
	err := ctl.DB.QueryRow(c, `SELECT p.id,p.group_id,p.title,p.pdf_url,p.time_limit_ms,p.memory_limit_mb,p.first_blood_user_id,g.start_time,g.end_time FROM problems p JOIN problem_groups g ON g.id=p.group_id WHERE p.id=$1`, c.Param("id")).Scan(&p.ID, &p.GroupID, &p.Title, &p.PDFURL, &p.TimeLimitMS, &p.MemoryLimitMB, &p.FirstBloodUserID, &start, &end)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Soal tidak ditemukan", nil)
		} else {
			response.Fail(c, 400, "VALIDATION_ERROR", "ID soal tidak valid", nil)
		}
		return
	}
	now := time.Now()
	if !isAdmin(c) && (now.Before(start) || now.After(end)) {
		response.Fail(c, 403, "CONTEST_CLOSED", "Kontes belum dibuka atau sudah ditutup", nil)
		return
	}
	if p.FirstBloodUserID != nil {
		var fb model.FirstBloodUser
		e := ctl.DB.QueryRow(c, `SELECT id,username,name FROM users WHERE id=$1`, *p.FirstBloodUserID).Scan(&fb.ID, &fb.Username, &fb.Name)
		if e == nil {
			p.FirstBloodUser = &fb
		}
	}
	rows, e := ctl.DB.Query(c, `SELECT id,input_data,expected_output,is_sample FROM test_cases WHERE problem_id=$1 AND is_sample=TRUE ORDER BY id`, p.ID)
	if e != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal mengambil sample test case", nil)
		return
	}
	defer rows.Close()
	samples := []model.TestCase{}
	for rows.Next() {
		var t model.TestCase
		if rows.Scan(&t.ID, &t.InputData, &t.ExpectedOutput, &t.IsSample) == nil {
			samples = append(samples, t)
		}
	}
	response.OK(c, 200, "Detail soal berhasil didapatkan", gin.H{"id": p.ID, "group_id": p.GroupID, "title": p.Title, "pdf_url": p.PDFURL, "time_limit_ms": p.TimeLimitMS, "memory_limit_mb": p.MemoryLimitMB, "first_blood_user": p.FirstBloodUser, "sample_test_cases": samples})
}
func (ctl *Controller) AddTestCase(c *gin.Context) {
	var in testCaseInput
	if c.ShouldBindJSON(&in) != nil {
		response.Fail(c, 400, "VALIDATION_ERROR", "Input test case tidak valid", nil)
		return
	}
	var t model.TestCase
	err := ctl.DB.QueryRow(c, `INSERT INTO test_cases(problem_id,input_data,expected_output,is_sample) VALUES($1,$2,$3,$4) RETURNING id,problem_id,input_data,expected_output,is_sample`, c.Param("id"), in.InputData, in.ExpectedOutput, in.IsSample).Scan(&t.ID, &t.ProblemID, &t.InputData, &t.ExpectedOutput, &t.IsSample)
	if err != nil {
		response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Soal tidak ditemukan atau ID tidak valid", nil)
		return
	}
	response.OK(c, 201, "Test case berhasil ditambahkan", t)
}
