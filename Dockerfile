FROM golang:1.25.5 AS base
WORKDIR /pay
COPY go.mod go.sum ./
RUN go mod download
COPY . .


FROM base AS dev
CMD [ "go", "run", "main.go" ]


FROM base AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go


FROM alpine:latest AS certs
RUN apk --update add ca-certificates


FROM scratch
COPY --from=builder /pay/main /main
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
CMD [ "/main" ]
