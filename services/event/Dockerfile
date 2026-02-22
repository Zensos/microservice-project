FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /event ./services/event

FROM alpine:3.21
COPY --from=builder /event /event
EXPOSE 3002
CMD ["/event"]
