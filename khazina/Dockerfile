FROM golang:1.25.9 AS base
WORKDIR /khazina

RUN go install github.com/go-delve/delve/cmd/dlv@v1.26.1

COPY go.mod go.sum ./
RUN go mod download
COPY . .


FROM base AS dev
CMD ["dlv", "debug", "main.go", "--listen=:40000", "--headless=true", "--api-version=2", "--accept-multiclient", "--continue", "--build-flags=-gcflags='all=-N -l'"]


FROM base AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go


FROM alpine:latest AS certs
RUN apk --update add ca-certificates


FROM scratch
COPY --from=builder /khazina/main /main
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD [ "/main" ]
