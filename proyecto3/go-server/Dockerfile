FROM golang:latest AS builder
WORKDIR /app
# Copiamos la carpeta proto y go-server al contexto de compilación
COPY proto/ ./proto/
COPY go-server/ ./go-server/
WORKDIR /app/go-server
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/go-server/main .
EXPOSE 50051
CMD ["./main"]