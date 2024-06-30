FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN apk add build-base && apk cache clean
ENV CGO_ENABLED=1
RUN go build -o ./ozon-task ./cmd/ozon-task/main.go


FROM alpine
WORKDIR /app
COPY --from=builder /app/ozon-task ./ozon-task
COPY --from=builder /app/internal/migrations ./internal/migrations
COPY --from=builder /app/internal/database ./internal/database
EXPOSE 8080
CMD ["./ozon-task"]