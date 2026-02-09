FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/server ./cmd/server
RUN go build -o /out/publisher ./cmd/publisher

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/server /app/server
COPY --from=builder /out/publisher /app/publisher
