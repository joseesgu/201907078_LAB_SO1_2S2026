# Manual Técnico - Proyecto 1: Sistemas Operativos 1

## 1. Requerimientos del Entorno

Para el desarrollo y despliegue de esta infraestructura, se utilizaron las siguientes tecnologías y herramientas:

- **Host:** Laptop con Linux Mint y soporte de virtualización KVM/QEMU activado.
- **Sistema Operativo de VMs:** Ubuntu Server 26.04 LTS.
- **Lenguaje de Programación:** Go (Golang).
- **Runtimes de Contenedores:** Docker, Podman y Containerd.
- **Registro de Imágenes:** Zot (Project Zot).
- **Control de Versiones:** Git y GitHub.

## 2. Arquitectura del Sistema e IPs

El entorno está distribuido en tres Máquinas Virtuales (VMs) en red local:

- **VM1 (IP 192.168.122.248):** Ejecuta Containerd. Aloja la API 1 (Puerto 8081) y API 2 (Puerto 8082).
- **VM2 (IP 192.168.122.87):** Ejecuta Podman. Aloja la API 3 (Puerto 8083).
- **VM3 (IP 192.168.122.224):** Ejecuta Docker. Aloja el contenedor del registro privado Zot (Puerto 5000).

## 3. Guía de Instalación y Configuración por Máquina

### 3.1. VM1: Configuración de Containerd

Instalación del runtime Containerd puro:

```bash
sudo apt update && sudo apt install containerd -y
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml
sudo systemctl restart containerd
sudo systemctl enable containerd
```

_Nota de automatización:_ Containerd no soporta reinicio automático de contenedores desde su cliente `ctr`. Se implementó un script `levantar_apis.sh` que limpia y ejecuta los contenedores en cada inicio de sesión.

### 3.2. VM2: Configuración de Podman

Instalación del runtime Podman y configuración para permitir conexiones inseguras (HTTP) al registro Zot:

```bash
sudo apt update && sudo apt install podman -y
sudo mkdir -p /etc/containers/registries.conf.d
sudo nano /etc/containers/registries.conf.d/zot.conf
```

_Contenido de zot.conf:_

```toml
[[registry]]
location = "192.168.122.224:5000"
insecure = true
```

### 3.3. VM3: Configuración de Docker y Zot

Instalación de Docker, habilitación de registros inseguros y ejecución de Zot con reinicio automático:

```bash
sudo apt update && sudo apt install docker.io -y
sudo systemctl enable docker

# Configuración de Insecure Registry

sudo nano /etc/docker/daemon.json

# (Añadir: { "insecure-registries": ["192.168.122.224:5000"] })

sudo systemctl restart docker

# Configuración y ejecución de Zot

mkdir -p ~/.zot
sudo nano ~/.zot/config.json

# (Configurar rootDirectory a /var/lib/registry y puerto 5000)

sudo docker run -d -p 5000:5000 --name zot-registry --restart always -v ~/.zot/config.json:/etc/zot/config.json ghcr.io/project-zot/zot-linux-amd64:latest
```

## 4. Construcción y Despliegue de APIs (Flujo de Trabajo)

El código de las APIs, desarrollado en Go y con Dockerfiles Multi-stage, se obtiene desde GitHub, se construye en la VM3 y se distribuye.

### Paso 1: Descargar código y construir (Ejecutado en VM3)

```bash

# Entrar a la carpeta y descargar los cambios

cd ~/201907078_LAB_SO1_2S2026
git pull

# Construir las 3 imágenes con la etiqueta v2

sudo docker build -t 192.168.122.224:5000/api1-201907078:v2 ./api1
sudo docker build -t 192.168.122.224:5000/api2-201907078:v2 ./api2
sudo docker build -t 192.168.122.224:5000/api3-201907078:v2 ./api3

# Subirlas al registro Zot

sudo docker push 192.168.122.224:5000/api1-201907078:v2
sudo docker push 192.168.122.224:5000/api2-201907078:v2
sudo docker push 192.168.122.224:5000/api3-201907078:v2
```

### Paso 2: Ejecución de Contenedores (VM1 y VM2)

**En VM2 (Podman):**

```bash
sudo podman run -d -p 8083:8083 --name api3_container --replace 192.168.122.224:5000/api3-201907078:v2
```

**En VM1 (Containerd):**

```bash
sudo ctr images pull --plain-http 192.168.122.224:5000/api1-201907078:v2
sudo ctr images pull --plain-http 192.168.122.224:5000/api2-201907078:v2
sudo ctr run --net-host -d 192.168.122.224:5000/api1-201907078:v2 api1_container
sudo ctr run --net-host -d 192.168.122.224:5000/api2-201907078:v2 api2_container

cat << 'EOF' > levantar_apis.sh
#!/bin/bash
echo "Limpiando contenedores antiguos..."
sudo ctr containers rm api1_container 2>/dev/null
sudo ctr containers rm api2_container 2>/dev/null

echo "Levantando API 1 y API 2..."
sudo ctr run --net-host -d 192.168.122.224:5000/api1-201907078:v2 api1_container
sudo ctr run --net-host -d 192.168.122.224:5000/api2-201907078:v2 api2_container
echo "¡APIs listas!"
EOF

chmod +x levantar_apis.sh

./levantar_apis.sh

```
