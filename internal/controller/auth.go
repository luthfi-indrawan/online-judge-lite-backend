package controller

import (
	"context"
	"strings"

	"online-judge-lite-backend/internal/model"
	"online-judge-lite-backend/internal/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ctl *Controller) Login(c *gin.Context) {
	var req loginRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, 400, "VALIDATION_ERROR", "Username dan password wajib diisi", nil)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	var id int
	var username, name, role, hash string
	err := ctl.DB.QueryRow(c, `SELECT id,username,name,role::text,password_hash FROM users WHERE username=$1`, req.Username).Scan(&id, &username, &name, &role, &hash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		response.Fail(c, 401, "UNAUTHORIZED", "Username atau password salah", nil)
		return
	}
	token, err := ctl.makeToken(id, username, role)
	if err != nil {
		response.Fail(c, 500, "INTERNAL_SERVER_ERROR", "Gagal membuat token", nil)
		return
	}
	response.OK(c, 200, "Login berhasil", gin.H{"token": token, "user": model.User{ID: id, Username: username, Name: name, Role: role}})
}
func (ctl *Controller) Me(c *gin.Context) {
	var u model.User
	err := ctl.DB.QueryRow(c, `SELECT id,username,name,role::text,created_at FROM users WHERE id=$1`, uid(c)).Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.CreatedAt)
	if err != nil {
		response.Fail(c, 404, "RESOURCE_NOT_FOUND", "Pengguna tidak ditemukan", nil)
		return
	}
	response.OK(c, 200, "Profil berhasil didapatkan", u)
}
func (ctl *Controller) SeedAdmin(ctx context.Context) error {
	if strings.TrimSpace(ctl.Config.AdminPassword) == "" {
		return nil
	}
	var exists bool
	err := ctl.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`, ctl.Config.AdminUsername).Scan(&exists)
	if err != nil || exists {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(ctl.Config.AdminPassword), 10)
	if err != nil {
		return err
	}
	_, err = ctl.DB.Exec(ctx, `INSERT INTO users(username,name,password_hash,role) VALUES($1,$2,$3,'ADMIN') ON CONFLICT(username) DO NOTHING`, ctl.Config.AdminUsername, ctl.Config.AdminName, string(hash))
	return err
}
