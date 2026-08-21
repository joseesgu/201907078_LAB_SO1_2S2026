# Guía de Pruebas y Evidencia Funcional

## Fase 1: Preparación del Entorno

- [ ] **Evidencia 1.1:** Captura de la ventana de `virt-manager` mostrando las 3 máquinas virtuales creadas y corriendo en KVM.
      ![Evidencia 1.1](image.png)
- [ ] **Evidencia 1.2:** Captura de la terminal de cada VM mostrando la versión de su respectivo runtime:
  - VM1: Resultado de ejecutar `sudo ctr version`.
    ![Evidencia 1.2](image-1.png)
  - VM2: Resultado de ejecutar `podman --version`.
    ![Evidencia 1.2](image-2.png)
  - VM3: Resultado de ejecutar `docker --version`.
    ![Evidencia 1.2](image-3.png)

## Fase 2: Registros y Comunicación Cruzada

- [ ] **Evidencia 2.1 (Registro Zot):** Captura ejecutando `curl http://192.168.122.224:5000/v2/_catalog` desde cualquier VM, donde se observe el JSON con la lista de los 3 repositorios (api1, api2, api3) almacenados en Zot.
      ![Evidencia 2.1](image-8.png)
- [ ] **Evidencia 2.2 (Push/Pull):** Captura de la VM1 y VM2 realizando la descarga de imágenes.
  - VM1: Resultado de `sudo ctr images ls` mostrando las imágenes de Zot, API 1 y API2.
    ![Evidencia 2.2](image-4.png)
  - VM2: Resultado de `sudo podman images` mostrando la imagen de la API 3.
    ![Evidencia 2.2](image-5.png)

## Fase 3: Desarrollo de APIs y Endpoint /health

- [ ] **Evidencia 3.1 (Pruebas de Salud):** Capturas realizando peticiones directas al endpoint `/health` de cada API, evidenciando el JSON con los campos requeridos (`status`, `timestamp`, `VM`, `carnet`).
  - `curl http://192.168.122.248:8081/health`
    ![Evidencia 3.1](image-6.png)
  - `curl http://192.168.122.248:8082/health`
    ![Evidencia 3.1](image-7.png)
  - `curl http://192.168.122.87:8083/health`
    ![Evidencia 3.1](image-9.png)

## Fase 4: Validación Final - Comunicación Cruzada

- [ ] **Evidencia 4.1 (API 1):** Captura ejecutando:
  - `curl http://192.168.122.248:8081/api1/201907078/call-api3`
  - Validar que devuelva el JSON con `"connection": true` y el mensaje correcto.
    ![Evidencia 4.1](image-10.png)
- [ ] **Evidencia 4.2 (API 2):** Captura ejecutando:
  - `curl http://192.168.122.248:8082/api2/201907078/call-api1`
    ![Evidencia 4.2](image-11.png)
- [ ] **Evidencia 4.3 (API 3):** Captura ejecutando:
  - `curl http://192.168.122.87:8083/api3/201907078/call-api2`
    ![Evidencia 4.3](image-12.png)
- [ ] **Evidencia 4.4 (Manejo de Errores):** Apagar momentáneamente la API 3 y ejecutar la Evidencia 4.1 para capturar el JSON demostrando `"connection": false` y el manejo correcto del error HTTP.
      ![Evidencia 4.4](image-13.png)
