package model

import "time"

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
type Group struct {
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Status        string    `json:"status"`
	TotalProblems int       `json:"total_problems"`
	CreatedBy     *int      `json:"created_by"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}
type Problem struct {
	ID               int             `json:"id"`
	GroupID          int             `json:"group_id"`
	Title            string          `json:"title"`
	PDFURL           string          `json:"pdf_url"`
	TimeLimitMS      int             `json:"time_limit_ms"`
	MemoryLimitMB    int             `json:"memory_limit_mb"`
	FirstBloodUserID *int            `json:"first_blood_user_id,omitempty"`
	FirstBloodUser   *FirstBloodUser `json:"first_blood_user,omitempty"`
	UserStatus       string          `json:"user_status,omitempty"`
	CreatedAt        time.Time       `json:"created_at,omitempty"`
}
type FirstBloodUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}
type TestCase struct {
	ID             int    `json:"id"`
	ProblemID      int    `json:"problem_id,omitempty"`
	InputData      string `json:"input_data"`
	ExpectedOutput string `json:"expected_output"`
	IsSample       bool   `json:"is_sample"`
}
type Submission struct {
	ID              int       `json:"id"`
	UserID          int       `json:"user_id"`
	ProblemID       int       `json:"problem_id"`
	Language        string    `json:"language"`
	Code            string    `json:"code,omitempty"`
	Verdict         string    `json:"verdict"`
	ExecutionTimeMS int       `json:"execution_time_ms"`
	CompileOutput   string    `json:"compile_output,omitempty"`
	SubmittedAt     time.Time `json:"submitted_at"`
}
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
