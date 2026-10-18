package controller

import (
	"online-judge-lite-backend/internal/model"
	"online-judge-lite-backend/internal/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type groupInput struct {
	Title       string    `json:"title" binding:"required"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
}

func statusFor(start, end, now time.Time) string {
	if now.Before(start) {
		return "UPCOMING"
	}
	if now.After(end) {
		return "ENDED"
	}
	return "ONGOING"
}
func (ctl *Controller) ListGroups(c *gin.Context) {
	rows, err := ctl.DB.Query(c, `SELECT g.id,g.title,COALESCE(g.description,''),g.start_time,g.end_time,g.created_by,g.created_at,COUNT(p.id)::int FROM problem_groups g LEFT JOIN problems p ON p.group_id=g.id GROUP BY g.id ORDER BY g.start_time DESC`)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal mengambil grup soal", nil)
		return
	}
	defer rows.Close()
	items := []model.Group{}
	now := time.Now()
	for rows.Next() {
		var g model.Group
		if rows.Scan(&g.ID, &g.Title, &g.Description, &g.StartTime, &g.EndTime, &g.CreatedBy, &g.CreatedAt, &g.TotalProblems) == nil {
			g.Status = statusFor(g.StartTime, g.EndTime, now)
			items = append(items, g)
		}
	}
	response.OK(c, 200, "Daftar grup soal berhasil didapatkan", items)
}
func (ctl *Controller) GetGroup(c *gin.Context) {
	var g model.Group
	err := ctl.DB.QueryRow(c, `SELECT id,title,COALESCE(description,''),start_time,end_time,created_by,created_at,(SELECT COUNT(*)::int FROM problems WHERE group_id=problem_groups.id) FROM problem_groups WHERE id=$1`, c.Param("id")).Scan(&g.ID, &g.Title, &g.Description, &g.StartTime, &g.EndTime, &g.CreatedBy, &g.CreatedAt, &g.TotalProblems)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Grup soal tidak ditemukan", nil)
		} else {
			response.Fail(c, 400, "VALIDATION_ERROR", "ID grup tidak valid", nil)
		}
		return
	}
	g.Status = statusFor(g.StartTime, g.EndTime, time.Now())
	if !isAdmin(c) && g.Status == "UPCOMING" {
		response.Fail(c, 403, "FORBIDDEN_ACCESS", "Grup soal belum dapat diakses", nil)
		return
	}
	rows, e := ctl.DB.Query(c, `SELECT p.id,p.group_id,p.title,p.pdf_url,p.time_limit_ms,p.memory_limit_mb,p.first_blood_user_id,p.created_at,CASE WHEN EXISTS(SELECT 1 FROM submissions s WHERE s.problem_id=p.id AND s.user_id=$2 AND s.verdict='AC') AND p.first_blood_user_id=$2 THEN 'FIRST_BLOOD' WHEN EXISTS(SELECT 1 FROM submissions s WHERE s.problem_id=p.id AND s.user_id=$2 AND s.verdict='AC') THEN 'AC' WHEN EXISTS(SELECT 1 FROM submissions s WHERE s.problem_id=p.id AND s.user_id=$2) THEN 'ATTEMPTED' ELSE 'NOT_ATTEMPTED' END FROM problems p WHERE p.group_id=$1 ORDER BY p.id`, g.ID, uid(c))
	if e != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal mengambil daftar soal", nil)
		return
	}
	defer rows.Close()
	problems := []model.Problem{}
	for rows.Next() {
		var p model.Problem
		if rows.Scan(&p.ID, &p.GroupID, &p.Title, &p.PDFURL, &p.TimeLimitMS, &p.MemoryLimitMB, &p.FirstBloodUserID, &p.CreatedAt, &p.UserStatus) == nil {
			problems = append(problems, p)
		}
	}
	response.OK(c, 200, "Detail grup soal berhasil didapatkan", gin.H{"id": g.ID, "title": g.Title, "description": g.Description, "start_time": g.StartTime, "end_time": g.EndTime, "status": g.Status, "problems": problems})
}
func (ctl *Controller) CreateGroup(c *gin.Context) {
	var in groupInput
	if c.ShouldBindJSON(&in) != nil || in.Title == "" || !in.EndTime.After(in.StartTime) {
		response.Fail(c, 400, "VALIDATION_ERROR", "Judul dan waktu valid wajib diisi; end_time harus setelah start_time", nil)
		return
	}
	var g model.Group
	err := ctl.DB.QueryRow(c, `INSERT INTO problem_groups(title,description,start_time,end_time,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,title,COALESCE(description,''),start_time,end_time,created_by,created_at`, in.Title, in.Description, in.StartTime, in.EndTime, uid(c)).Scan(&g.ID, &g.Title, &g.Description, &g.StartTime, &g.EndTime, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal membuat grup soal", nil)
		return
	}
	response.OK(c, 201, "Grup soal berhasil dibuat", g)
}
