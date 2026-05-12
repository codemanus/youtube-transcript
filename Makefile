.PHONY: build build-frontend sync-static build-backend run-backend release-linux release-linux-arm64 release-for-host print-release-binary pull update update-linux update-release-host

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

release-linux-arm64: build-frontend sync-static
	mkdir -p dist
	cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../dist/youtube-transcript-linux-arm64 .

# Linux binary matching this machine (amd64 or arm64). Use on LXC after git pull.
release-for-host: build-frontend sync-static
	@mkdir -p dist
	@arch=$$(uname -m); \
	case $$arch in \
		x86_64) goarch=amd64 ;; \
		aarch64|arm64) goarch=arm64 ;; \
		*) echo "unsupported uname -m: $$arch (expected x86_64 or aarch64)"; exit 1 ;; \
	esac; \
	echo "==> GOOS=linux GOARCH=$$goarch"; \
	cd backend && GOOS=linux GOARCH=$$goarch CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../dist/youtube-transcript-linux-$$goarch .

# Print path to the host-target Linux binary (for scripts). Requires prior release-for-host.
print-release-binary:
	@arch=$$(uname -m); \
	case $$arch in \
		x86_64) goarch=amd64 ;; \
		aarch64|arm64) goarch=arm64 ;; \
		*) echo "unsupported uname -m: $$arch" >&2; exit 1 ;; \
	esac; \
	echo "dist/youtube-transcript-linux-$$goarch"

pull:
	git pull --ff-only

# Workstation: pull + full native build (UI + embed + local binary).
update: pull build

# Linux host/LXC: pull + linux/amd64 release (override with release-linux-arm64 on arm64 if not using release-for-host).
update-linux: pull release-linux

# Linux host/LXC: pull + binary for this CPU (amd64 or arm64).
update-release-host: pull release-for-host

run-backend:
	cd backend && LISTEN_ADDR=:8080 go run .
