FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

# Run tests with race detector enabled
RUN CGO_ENABLED=1 GOOS=linux go test -v -race ./... 

RUN CGO_ENABLED=0 GOOS=linux go build -o backend ./src/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Bangkok

WORKDIR /root/

COPY --from=builder /app/backend .