# Etapa 1: Construcción (Builder)
FROM golang:1.20-alpine AS builder

# Establecemos el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copiamos el código fuente al contenedor
COPY main.go .

# Compilamos la aplicación de Go
# CGO_ENABLED=0 asegura que el binario sea estático y funcione en imágenes scratch/alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o apiserver main.go

# Etapa 2: Imagen Final Ligera
FROM alpine:latest

# Añadimos certificados raíz por si la API necesita hacer peticiones seguras (opcional pero recomendado)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copiamos el binario compilado desde la etapa anterior
COPY --from=builder /app/apiserver .

# Exponemos el puerto (Cámbialo a 8082 o 8083 según la API que estés construyendo)
EXPOSE 8081

# Comando para ejecutar la aplicación
CMD ["./apiserver"]