FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /payment ./services/payment

FROM alpine:3.21
COPY --from=builder /payment /payment
EXPOSE 3004
CMD ["/payment"]
