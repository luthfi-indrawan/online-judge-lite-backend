package controller

import (
	"net/http"
	"strings"

	"online-judge-lite-backend/internal/response"
	"online-judge-lite-backend/internal/runner"

	"github.com/gin-gonic/gin"
)

type runCodeInput struct {
	Language string `json:"language" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Input    string `json:"input"`
}

func (ctl *Controller) RunCode(c *gin.Context) {
	// Batasi ukuran request JSON secara keseluruhan.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 300*1024)

	var in runCodeInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Request JSON tidak valid atau ukurannya terlalu besar", nil)
		return
	}

	in.Language = strings.TrimSpace(in.Language)
	if in.Language != "c" && in.Language != "cpp" {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Language harus berupa c atau cpp", nil)
		return
	}
	if strings.TrimSpace(in.Code) == "" || len(in.Code) > 200*1024 {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Code wajib diisi dan maksimal 200 KB", nil)
		return
	}
	if len(in.Input) > 32*1024 {
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Input maksimal 32 KB", nil)
		return
	}

	result := ctl.Runner.Run(c.Request.Context(), runner.Request{
		Language: in.Language,
		Code:     in.Code,
		Input:    in.Input,
	})

	switch result.Status {
	case "SUCCESS":
		response.OK(c, http.StatusOK, "Program berhasil dijalankan", result)
	case "COMPILE_ERROR":
		response.OK(c, http.StatusOK, "Program gagal dikompilasi", result)
	case "TIME_LIMIT_EXCEEDED":
		response.OK(c, http.StatusOK, "Program melewati batas waktu", result)
	case "RUNTIME_ERROR":
		response.OK(c, http.StatusOK, "Terjadi runtime error", result)
	case "COMPILE_TIMEOUT":
		response.OK(c, http.StatusOK, "Kompilasi melewati batas waktu", result)
	case "INVALID_SOURCE", "INVALID_INPUT", "INVALID_LANGUAGE":
		response.Fail(c, http.StatusBadRequest, "VALIDATION_ERROR",
			"Source code, bahasa, atau input tidak valid", nil)
	case "CANCELLED":
		// Klien dapat menutup koneksi saat program sedang berjalan.
		return
	default:
		response.Fail(c, http.StatusInternalServerError, "RUN_CODE_ERROR",
			"Backend gagal menjalankan program", nil)
	}
}
