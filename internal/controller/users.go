package controller

import (
	"fmt"
	"strings"

	"online-judge-lite-backend/internal/model"
	"online-judge-lite-backend/internal/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type userInput struct {
	Username string `json:"username" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role"`
}
type bulkUsersRequest struct {
	Users []userInput `json:"users" binding:"required,min=1,max=500"`
}

func (ctl *Controller) ListUsers(c *gin.Context) {
	rows, err := ctl.DB.Query(c, `SELECT id,username,name,role::text,created_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal mengambil pengguna", nil)
		return
	}
	defer rows.Close()
	users := []model.User{}
	for rows.Next() {
		var u model.User
		if rows.Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.CreatedAt) == nil {
			users = append(users, u)
		}
	}
	response.OK(c, 200, "Daftar pengguna berhasil didapatkan", users)
}
func validateUser(in *userInput) string {
	in.Username = strings.TrimSpace(in.Username)
	in.Name = strings.TrimSpace(in.Name)
	if in.Username == "" || in.Name == "" || len(in.Username) > 50 || len(in.Name) > 100 || len(in.Password) < 8 {
		return "Username, nama, atau password tidak valid (password minimal 8 karakter)"
	}
	if in.Role == "" {
		in.Role = "USER"
	}
	if in.Role != "USER" && in.Role != "ADMIN" {
		return "Role harus USER atau ADMIN"
	}
	return ""
}
func (ctl *Controller) CreateUser(c *gin.Context) {
	var in userInput
	if c.ShouldBindJSON(&in) != nil || validateUser(&in) != "" {
		response.Fail(c, 400, "VALIDATION_ERROR", "Data pengguna tidak valid; password minimal 8 karakter", nil)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 10)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal memproses password", nil)
		return
	}
	var u model.User
	err = ctl.DB.QueryRow(c, `INSERT INTO users(username,name,password_hash,role) VALUES($1,$2,$3,$4::user_role) RETURNING id,username,name,role::text,created_at`, in.Username, in.Name, string(hash), in.Role).Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.CreatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			response.Fail(c, 409, "DUPLICATE_ENTRY", "Username sudah digunakan", nil)
		} else {
			response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal membuat pengguna", nil)
		}
		return
	}
	response.OK(c, 201, "Pengguna berhasil dibuat", u)
}
func (ctl *Controller) BulkCreateUsers(c *gin.Context) {
	var req bulkUsersRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, 400, "VALIDATION_ERROR", "Array users wajib diisi (maksimal 500 akun)", nil)
		return
	}
	tx, err := ctl.DB.Begin(c)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal memulai transaksi", nil)
		return
	}
	defer tx.Rollback(c)
	out := []gin.H{}
	seen := map[string]bool{}
	for i := range req.Users {
		in := req.Users[i]
		if msg := validateUser(&in); msg != "" {
			response.Fail(c, 400, "VALIDATION_ERROR", fmt.Sprintf("%s pada baris %d", msg, i+1), nil)
			return
		}
		if seen[in.Username] {
			response.Fail(c, 409, "DUPLICATE_ENTRY", "Username duplikat di dalam file import: "+in.Username, nil)
			return
		}
		seen[in.Username] = true
		hash, e := bcrypt.GenerateFromPassword([]byte(in.Password), 10)
		if e != nil {
			response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal memproses password", nil)
			return
		}
		var id int
		var username, name string
		e = tx.QueryRow(c, `INSERT INTO users(username,name,password_hash,role) VALUES($1,$2,$3,$4::user_role) RETURNING id,username,name`, in.Username, in.Name, string(hash), in.Role).Scan(&id, &username, &name)
		if e != nil {
			if strings.Contains(strings.ToLower(e.Error()), "unique") || strings.Contains(strings.ToLower(e.Error()), "duplicate") {
				response.Fail(c, 409, "DUPLICATE_ENTRY", "Username sudah digunakan: "+in.Username, nil)
			} else {
				response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal membuat akun secara bulk", nil)
			}
			return
		}
		out = append(out, gin.H{"id": id, "username": username, "name": name})
	}
	if err = tx.Commit(c); err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal menyimpan bulk import", nil)
		return
	}
	response.OK(c, 201, "Bulk create pengguna berhasil", gin.H{"created_count": len(out), "users": out})
}
func (ctl *Controller) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, 400, "VALIDATION_ERROR", "new_password wajib diisi (minimal 8 karakter)", nil)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal memproses password", nil)
		return
	}
	tag, err := ctl.DB.Exec(c, `UPDATE users SET password_hash=$1 WHERE id=$2`, string(hash), id)
	if err != nil {
		response.Fail(c, 400, "VALIDATION_ERROR", "ID pengguna tidak valid", nil)
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Pengguna tidak ditemukan", nil)
		return
	}
	response.OK(c, 200, "Password pengguna berhasil diperbarui", nil)
}
