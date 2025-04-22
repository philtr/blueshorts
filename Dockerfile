# build stage
FROM golang:1.24.2-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o blueshorts ./cmd/blueshorts

# runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /src/blueshorts .
ENTRYPOINT ["./blueshorts"]
