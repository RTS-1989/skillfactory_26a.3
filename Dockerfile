FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine:latest
LABEL version="1.0.0"
WORKDIR /app
COPY --from=builder /app/main /app/main
CMD ["./main"]