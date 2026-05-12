.PHONY: build build-frontend sync-static build-backend run-backend release-linux

build: build-frontend sync-static build-backend

build-frontend:
	cd frontend && npm ci && npm run build

sync-static:
	rm -rf backend/static/*
	mkdir -p backend/static
	cp -a frontend/build/. backend/static/

build-backend:
	mkdir -p dist
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../dist/youtube-transcript .

release-linux: build-frontend sync-static
	mkdir -p dist
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../dist/youtube-transcript-linux-amd64 .

run-backend:
	cd backend && LISTEN_ADDR=:8080 go run .
