package controller

import (
	"online-judge-lite-backend/internal/response"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (ctl *Controller) Scoreboard(c *gin.Context) {
	groupID := c.Param("group_id")
	groupIDInt, parseErr := strconv.Atoi(groupID)
	if parseErr != nil || groupIDInt < 1 {
		response.Fail(c, 400, "VALIDATION_ERROR", "ID grup tidak valid", nil)
		return
	}
	var title string
	err := ctl.DB.QueryRow(c, `SELECT title FROM problem_groups WHERE id=$1`, groupID).Scan(&title)
	if err != nil {
		response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Grup soal tidak ditemukan", nil)
		return
	}
	prows, err := ctl.DB.Query(c, `SELECT id,title,first_blood_user_id FROM problems WHERE group_id=$1 ORDER BY id`, groupID)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal mengambil daftar soal", nil)
		return
	}
	type prob struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		First *int
	}
	problems := []prob{}
	for prows.Next() {
		var p prob
		if prows.Scan(&p.ID, &p.Title, &p.First) == nil {
			problems = append(problems, p)
		}
	}
	prows.Close()
	rows, err := ctl.DB.Query(c, `SELECT u.id,u.username,u.name,p.id,s.verdict::text,extract(epoch from s.submitted_at)::bigint,p.first_blood_user_id FROM users u CROSS JOIN problems p LEFT JOIN submissions s ON s.user_id=u.id AND s.problem_id=p.id WHERE u.role='USER' AND p.group_id=$1 ORDER BY u.id,s.submitted_at`, groupID)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal menghitung scoreboard", nil)
		return
	}
	defer rows.Close()
	type cell struct {
		Verdict      string `json:"verdict"`
		IsFirstBlood bool   `json:"is_first_blood"`
		ColorCode    string `json:"color_code"`
		AttemptCount int    `json:"attempt_count"`
	}
	type rankRow struct {
		UserID            int             `json:"user_id"`
		Username          string          `json:"username"`
		Name              string          `json:"name"`
		TotalSolved       int             `json:"total_solved"`
		SubmissionsStatus map[string]cell `json:"submissions_status"`
		acTime            int64
	}
	byUser := map[int]*rankRow{}
	order := []int{}
	for rows.Next() {
		var id, pid int
		var username, name string
		var verdict *string
		var submitted *int64
		var first *int
		if err := rows.Scan(&id, &username, &name, &pid, &verdict, &submitted, &first); err != nil {
			continue
		}
		r, ok := byUser[id]
		if !ok {
			r = &rankRow{UserID: id, Username: username, Name: name, SubmissionsStatus: map[string]cell{}}
			byUser[id] = r
			order = append(order, id)
		}
		if verdict == nil {
			continue
		}
		key := strconvI(pid)
		cc := r.SubmissionsStatus[key]
		cc.AttemptCount++
		if *verdict == "AC" {
			if cc.Verdict != "AC" {
				r.TotalSolved++
			}
			cc.Verdict = "AC"
			cc.IsFirstBlood = first != nil && *first == id
			cc.ColorCode = "LIGHT_GREEN"
			if cc.IsFirstBlood {
				cc.ColorCode = "DARK_GREEN"
			}
			if submitted != nil && (r.acTime == 0 || *submitted < r.acTime) {
				r.acTime = *submitted
			}
		} else if cc.Verdict != "AC" {
			cc.Verdict = *verdict
			cc.ColorCode = "RED"
		}
		r.SubmissionsStatus[key] = cc
	}
	for _, r := range byUser {
		for _, p := range problems {
			k := strconvI(p.ID)
			cc, ok := r.SubmissionsStatus[k]
			if !ok {
				cc = cell{Verdict: "NOT_ATTEMPTED", ColorCode: "GRAY"}
			}
			if cc.Verdict == "" {
				cc.Verdict = "NOT_ATTEMPTED"
				cc.ColorCode = "GRAY"
			}
			if p.First != nil && *p.First == r.UserID && cc.Verdict == "AC" {
				cc.IsFirstBlood = true
				cc.ColorCode = "DARK_GREEN"
			}
			r.SubmissionsStatus[k] = cc
		}
	}
	ranking := []gin.H{}
	for _, id := range order {
		r := byUser[id]
		ranking = append(ranking, gin.H{"user_id": r.UserID, "username": r.Username, "name": r.Name, "total_solved": r.TotalSolved, "submissions_status": r.SubmissionsStatus, "_ac_time": r.acTime})
	}
	sort.SliceStable(ranking, func(i, j int) bool {
		a := ranking[i]
		b := ranking[j]
		as := a["total_solved"].(int)
		bs := b["total_solved"].(int)
		if as != bs {
			return as > bs
		}
		at := a["_ac_time"].(int64)
		bt := b["_ac_time"].(int64)
		if at == 0 {
			return false
		}
		if bt == 0 {
			return true
		}
		return at < bt
	})
	out := []gin.H{}
	for i, r := range ranking {
		delete(r, "_ac_time")
		r["rank"] = i + 1
		out = append(out, r)
	}
	plist := []gin.H{}
	for _, p := range problems {
		plist = append(plist, gin.H{"id": p.ID, "title": p.Title})
	}
	response.OK(c, 200, "Scoreboard berhasil didapatkan", gin.H{"group_id": groupIDInt, "group_title": title, "problems": plist, "rankings": out})
}
func strconvI(i int) string { return strconv.Itoa(i) }
