FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /member ./services/member

FROM alpine:3.21
COPY --from=builder /member /member
EXPOSE 3003
CMD ["/member"]
