# Stage 1: Build
FROM golang:1.26-alpine AS builder

WORKDIR /build

# 필수 패키지만 설치
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 바이너리 빌드 (스트립해서 더 가볍게)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bot ./cmd/bot

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# 필요한 최소 패키지만 설치
RUN apk add --no-cache ca-certificates tzdata

# builder에서 바이너리만 복사
COPY --from=builder /build/bot .

ENTRYPOINT ["./bot"]
