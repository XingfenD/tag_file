FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o tagfile-webui ./cmd/webui

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /src/tagfile-webui /tagfile-webui

EXPOSE 8088

VOLUME ["/data"]

ENTRYPOINT ["/tagfile-webui", "--source", "/data"]
