CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ BEGIN CREATE TYPE user_role AS ENUM ('ADMIN', 'USER'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE programming_language AS ENUM ('c', 'cpp'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN CREATE TYPE submission_verdict AS ENUM ('PENDING', 'RUNNING', 'AC', 'WA', 'CE', 'TLE', 'RTE'); EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(50) UNIQUE NOT NULL,
  name VARCHAR(100) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  role user_role NOT NULL DEFAULT 'USER',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS problem_groups (
  id SERIAL PRIMARY KEY,
  title VARCHAR(150) NOT NULL,
  description TEXT,
  start_time TIMESTAMPTZ NOT NULL,
  end_time TIMESTAMPTZ NOT NULL,
  created_by INT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_group_time CHECK (end_time > start_time)
);
CREATE TABLE IF NOT EXISTS problems (
  id SERIAL PRIMARY KEY,
  group_id INT NOT NULL REFERENCES problem_groups(id) ON DELETE CASCADE,
  title VARCHAR(150) NOT NULL,
  pdf_url TEXT NOT NULL,
  time_limit_ms INT NOT NULL DEFAULT 1000 CHECK (time_limit_ms > 0),
  memory_limit_mb INT NOT NULL DEFAULT 128 CHECK (memory_limit_mb > 0),
  first_blood_user_id INT REFERENCES users(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS test_cases (
  id SERIAL PRIMARY KEY,
  problem_id INT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  input_data TEXT NOT NULL,
  expected_output TEXT NOT NULL,
  is_sample BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS submissions (
  id SERIAL PRIMARY KEY,
  user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  problem_id INT NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
  language programming_language NOT NULL,
  code TEXT NOT NULL,
  verdict submission_verdict NOT NULL DEFAULT 'PENDING',
  execution_time_ms INT NOT NULL DEFAULT 0,
  compile_output TEXT,
  submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_user ON submissions(problem_id, user_id, verdict);
CREATE INDEX IF NOT EXISTS idx_submissions_submitted_at ON submissions(submitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_problem_groups_schedule ON problem_groups(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_problems_group_id ON problems(group_id);
CREATE INDEX IF NOT EXISTS idx_test_cases_problem_id ON test_cases(problem_id);

CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = CURRENT_TIMESTAMP; RETURN NEW; END; $$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_problem_groups_updated_at ON problem_groups;
CREATE TRIGGER update_problem_groups_updated_at BEFORE UPDATE ON problem_groups FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
DROP TRIGGER IF EXISTS update_problems_updated_at ON problems;
CREATE TRIGGER update_problems_updated_at BEFORE UPDATE ON problems FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION check_first_blood() RETURNS TRIGGER AS $$
BEGIN
  IF NEW.verdict = 'AC' THEN
    UPDATE problems SET first_blood_user_id = NEW.user_id
    WHERE id = NEW.problem_id AND first_blood_user_id IS NULL;
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trigger_check_first_blood ON submissions;
CREATE TRIGGER trigger_check_first_blood AFTER INSERT OR UPDATE OF verdict ON submissions
FOR EACH ROW WHEN (NEW.verdict = 'AC') EXECUTE FUNCTION check_first_blood();
