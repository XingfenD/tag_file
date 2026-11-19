FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o tagfile-webui ./cmd/webui

FROM alpine:3.22

RUN apk add --no-cache ffmpeg ca-certificates tzdata

COPY --from=builder /src/tagfile-webui /tagfile-webui

EXPOSE 8088

VOLUME ["/data"]

ENTRYPOINT ["/tagfile-webui", "--source", "/data"]
