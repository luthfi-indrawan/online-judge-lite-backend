# Online Judge Lite — Backend

Backend REST API untuk Online Judge Lite, menggunakan **Go + Gin + PostgreSQL + JWT + Docker sandbox**. Struktur mengikuti permintaan `router -> handler -> controller`.

## 1. Stack

- Go 1.23+
- Gin HTTP router
- PostgreSQL 17 dan `pgx/v5`
- JWT HS256 dan bcrypt cost 10
- Worker pool asynchronous untuk submission
- Docker sandbox terisolasi untuk kompilasi/eksekusi C dan C++
- Supabase Storage dipakai langsung oleh frontend untuk PDF; backend menyimpan URL PDF

## 2. Struktur direktori

```text
online-judge-lite-backend/
├── cmd/api/main.go
├── internal/
│   ├── config/       # Environment configuration
│   ├── database/     # PostgreSQL connection pool
│   ├── model/        # API/domain data structures
│   ├── router/       # Route mapping and CORS
│   ├── handler/      # HTTP handler layer
│   ├── controller/   # Business logic and database operations
│   ├── middleware/   # JWT authentication and role access
│   ├── response/     # Standard JSON envelope
│   └── worker/       # Queue consumer and Docker sandbox runner
├── migrations/001_init.sql
├── sandbox/Dockerfile
├── .env.example
├── Dockerfile
└── docker-compose.yml
```

## 3. Run locally

Requirements: Go 1.23+, PostgreSQL, Docker Engine, and Docker CLI.

1. Create a PostgreSQL database named `online_judge` and run `migrations/001_init.sql`.
2. Copy `.env.example` to `.env`, then set `DATABASE_URL`, a random `JWT_SECRET` (at least 32 characters), and `ADMIN_PASSWORD`.
3. Build the sandbox image:

   ```bash
   docker build -t online-judge-sandbox:latest ./sandbox
   ```

4. Run the API:

   ```bash
   go mod tidy
   go run ./cmd/api
   ```

The API runs at `http://localhost:8080`; health check: `GET http://localhost:8080/health`. At startup, the API applies `migrations/001_init.sql` and creates the initial admin account if `ADMIN_PASSWORD` is set and the username does not already exist.

## 4. Run with Docker Compose

1. Copy `.env.example` to `.env` and update `JWT_SECRET` and `ADMIN_PASSWORD`.
2. Build the compiler image: `docker build -t online-judge-sandbox:latest ./sandbox`.
3. Start services: `docker compose up --build`.

The compose configuration uses `/tmp/online-judge-submissions` as the default host-shared workspace. You can override it with an **absolute host path** in `OJ_SANDBOX_HOST_DIR`. The same path must be mounted at the same location in the API container so the Docker daemon can mount each job workspace into its execution container.

> **Security warning:** mounting `/var/run/docker.sock` into the API container grants the API effective control over the host Docker daemon and can amount to host-root access. Use this only on a trusted, dedicated machine. For production, run the judge worker on a separate isolated host/service and keep the public API away from the Docker socket. Do not expose the Docker daemon or the API's Docker socket access to untrusted users.

## 5. Environment variables

| Variable                  | Purpose                                                | Default                          |
| ------------------------- | ------------------------------------------------------ | -------------------------------- |
| `APP_ENV`                 | `development` / `production`                           | `development`                    |
| `APP_PORT`                | HTTP port                                              | `8080`                           |
| `APP_BASE_PATH`           | API prefix                                             | `/api/v1`                        |
| `FRONTEND_ORIGINS`        | Comma-separated allowed origins                        | localhost:5173, localhost:3000   |
| `DATABASE_URL`            | PostgreSQL connection string                           | local postgres                   |
| `JWT_SECRET`              | HS256 signing secret, minimum 32 chars                 | development fallback; replace it |
| `JWT_TTL_HOURS`           | JWT lifetime                                           | `24`                             |
| `ADMIN_USERNAME`          | Initial admin username                                 | `admin`                          |
| `ADMIN_NAME`              | Initial admin display name                             | `System Administrator`           |
| `ADMIN_PASSWORD`          | Initial admin password; leave unset to disable seeding | unset                            |
| `WORKER_COUNT`            | Number of worker goroutines                            | `2`                              |
| `SUBMISSION_QUEUE_SIZE`   | In-memory queue capacity                               | `100`                            |
| `DOCKER_BIN`              | Docker CLI executable                                  | `docker`                         |
| `SANDBOX_IMAGE`           | Compiler image tag                                     | `online-judge-sandbox:latest`    |
| `SANDBOX_WORKDIR`         | Shared absolute workspace path                         | `./tmp/submissions`              |
| `COMPILE_TIMEOUT_SECONDS` | Maximum compile duration                               | `10`                             |

## 6. API endpoints

All JSON responses follow the envelope in `API_CONTRACT.md`. Protected endpoints require `Authorization: Bearer <token>`.

| Method | Endpoint                                 | Access                                              |
| ------ | ---------------------------------------- | --------------------------------------------------- |
| `POST` | `/api/v1/auth/login`                     | Public                                              |
| `GET`  | `/api/v1/auth/me`                        | Authenticated                                       |
| `GET`  | `/api/v1/admin/users`                    | Admin (extra endpoint for frontend listing)         |
| `POST` | `/api/v1/admin/users`                    | Admin                                               |
| `POST` | `/api/v1/admin/users/bulk`               | Admin                                               |
| `PUT`  | `/api/v1/admin/users/:id/reset-password` | Admin                                               |
| `GET`  | `/api/v1/problem-groups`                 | Authenticated                                       |
| `GET`  | `/api/v1/problem-groups/:id`             | Authenticated                                       |
| `POST` | `/api/v1/problem-groups`                 | Admin                                               |
| `POST` | `/api/v1/problems`                       | Admin                                               |
| `GET`  | `/api/v1/problems/:id`                   | Authenticated, contest window for students          |
| `POST` | `/api/v1/problems/:id/test-cases`        | Admin                                               |
| `POST` | `/api/v1/submissions`                    | Authenticated, contest window only                  |
| `GET`  | `/api/v1/submissions/me`                 | Authenticated (extra endpoint for frontend history) |
| `GET`  | `/api/v1/submissions/:id`                | Owner or Admin                                      |
| `GET`  | `/api/v1/scoreboards/groups/:group_id`   | Authenticated                                       |

## 7. Submission execution

1. The API validates the contest window and creates a `PENDING` submission.
2. The submission ID is placed in the in-process queue and the API returns `202 Accepted`.
3. A worker claims the job, loads **all** test cases, and compiles C/C++ in a disposable Docker container.
4. Each case runs in a container with `--network none`, memory/CPU/PID limits, dropped capabilities, `no-new-privileges`, and a read-only root filesystem. The workspace is the only writable bind mount.
5. Verdicts: `AC`, `WA`, `CE`, `TLE`, or `RTE`. The database trigger assigns First Blood atomically to the first accepted user.

The queue is in-memory, not a durable message broker. Pending jobs are periodically requeued after an API restart, but production deployments should move queueing to a durable broker and run workers independently. Sandbox limits and Docker isolation should be reviewed and hardened before running arbitrary code on a production host.

## 8. API contract notes

- Admin-created passwords are hashed using bcrypt cost 10 and are never returned.
- `GET /problems/:id` only returns test cases where `is_sample = TRUE`; hidden test cases remain server-side.
- `pdf_url` must be an HTTPS Supabase Storage object URL.
- Bulk user creation is transactional: a duplicate/invalid row rolls back the whole batch.
- Student submission history is owner-scoped. Admins may inspect any submission.
- Group status is computed dynamically as `UPCOMING`, `ONGOING`, or `ENDED` using server time.
- `GET /admin/users` and `GET /submissions/me` are convenience endpoints added for frontend use; they are not defined in the supplied API Contract.

## 9. Frontend integration

Set this in the frontend `.env`:

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

For a deployed frontend, add its exact origin to `FRONTEND_ORIGINS`. Do not use `*` with credentialed browser requests.
