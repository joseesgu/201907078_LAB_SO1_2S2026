FROM golang:latest AS builder
WORKDIR /app
# Copiamos la carpeta proto y go-client al contexto de compilación
COPY proto/ ./proto/
COPY go-client/ ./go-client/
WORKDIR /app/go-client
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/go-client/main .
EXPOSE 8081
CMD ["./main"]