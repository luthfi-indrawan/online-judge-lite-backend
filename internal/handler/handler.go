package handler

import (
	"online-judge-lite-backend/internal/controller"

	"github.com/gin-gonic/gin"
)

type Handler struct{ C *controller.Controller }

func New(c *controller.Controller) *Handler         { return &Handler{C: c} }
func (h *Handler) Login(c *gin.Context)             { h.C.Login(c) }
func (h *Handler) Me(c *gin.Context)                { h.C.Me(c) }
func (h *Handler) ListUsers(c *gin.Context)         { h.C.ListUsers(c) }
func (h *Handler) CreateUser(c *gin.Context)        { h.C.CreateUser(c) }
func (h *Handler) BulkCreateUsers(c *gin.Context)   { h.C.BulkCreateUsers(c) }
func (h *Handler) ResetPassword(c *gin.Context)     { h.C.ResetPassword(c) }
func (h *Handler) ListGroups(c *gin.Context)        { h.C.ListGroups(c) }
func (h *Handler) GetGroup(c *gin.Context)          { h.C.GetGroup(c) }
func (h *Handler) CreateGroup(c *gin.Context)       { h.C.CreateGroup(c) }
func (h *Handler) CreateProblem(c *gin.Context)     { h.C.CreateProblem(c) }
func (h *Handler) GetProblem(c *gin.Context)        { h.C.GetProblem(c) }
func (h *Handler) AddTestCase(c *gin.Context)       { h.C.AddTestCase(c) }
func (h *Handler) CreateSubmission(c *gin.Context)  { h.C.CreateSubmission(c) }
func (h *Handler) GetSubmission(c *gin.Context)     { h.C.GetSubmission(c) }
func (h *Handler) ListMySubmissions(c *gin.Context) { h.C.ListMySubmissions(c) }
func (h *Handler) Scoreboard(c *gin.Context)        { h.C.Scoreboard(c) }
