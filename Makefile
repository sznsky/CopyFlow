.PHONY: db api worker frontend build

# 可选：用 Docker 启动 MySQL（本地已有 MySQL 可跳过）
db:
	cd backend && docker compose up -d

# 启动 API（需本地 MySQL 已配置好）
api:
	cd backend && go run ./cmd/api

# 启动 Worker
worker:
	cd backend && go run ./cmd/worker

# 启动前端
frontend:
	cd frontend && npm run dev

# 编译后端
build:
	cd backend && go build -o bin/api.exe ./cmd/api && go build -o bin/worker.exe ./cmd/worker

# 编译前端
build-frontend:
	cd frontend && npm run build
