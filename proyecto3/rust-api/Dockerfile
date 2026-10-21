# Etapa de construcción
FROM rust:latest AS builder
WORKDIR /usr/src/app
# Instalar dependencias necesarias para compilar reqwest/openssl
RUN apt-get update && apt-get install -y pkg-config libssl-dev
COPY . .
RUN cargo build --release

# Etapa de ejecución (imagen final ligera)
FROM debian:bookworm-slim
# Instalar librerías de OpenSSL para que el binario pueda ejecutarse
RUN apt-get update && apt-get install -y openssl ca-certificates && rm -rf /var/lib/apt/lists/*
# Copiar el binario compilado desde la etapa anterior
COPY --from=builder /usr/src/app/target/release/rust-api /usr/local/bin/rust-api

EXPOSE 8080
CMD ["rust-api"]