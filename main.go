package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Constantes personalizadas
const CARNET = "201907078"
const VM_NUMBER = 1

// Estructura para el endpoint /health
type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        int    `json:"VM"`
	Carnet    string `json:"carnet"`
}

// Estructura para las respuestas de llamadas entre APIs
type CallResponse struct {
	Apiname    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

// Endpoint Principal: /health
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	resp := HealthResponse{
		Status:    "UP",
		Message:   "API1 is Ready",
		Timestamp: time.Now().Format(time.RFC3339),
		VM:        VM_NUMBER,
		Carnet:    CARNET,
	}
	
	json.NewEncoder(w).Encode(resp)
}

// Función genérica para consumir los /health de otras APIs
func callApiHandler(targetApi string, targetURL string, targetVM int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Petición HTTP al endpoint de la otra API
		resp, err := http.Get(targetURL + "/health")
		
		if err == nil && resp.StatusCode == http.StatusOK {
			var healthResp HealthResponse
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &healthResp)
			resp.Body.Close()

			// Validación de estado requerido en el manual
			if healthResp.Status == "UP" {
				successResp := CallResponse{
					Apiname:    targetApi,
					Message:    fmt.Sprintf("The %s located on the VM%d is working", targetApi, targetVM),
					Connection: true,
					Carnet:     CARNET,
				}
				json.NewEncoder(w).Encode(successResp)
				return
			}
		}

		// Si falla la conexión o el status no es "UP"
		errorResp := CallResponse{
			Apiname:    targetApi,
			Message:    fmt.Sprintf("ERROR: The %s located on the VM%d is not working", targetApi, targetVM),
			Connection: false,
			Carnet:     CARNET,
		}
		json.NewEncoder(w).Encode(errorResp)
	}
}

func main() {
	// IMPORTANTE: Sustituye <IP_DE_VM2> con la IP real de tu Máquina Virtual 2
	api2URL := "http://localhost:8082"            // API2 está en la misma VM1
	api3URL := "http://<IP_DE_VM2>:8083"          // API3 está en la VM2

	// Definición de Endpoints según el enunciado
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api1/201907078/call-api2", callApiHandler("API2", api2URL, 1))
	http.HandleFunc("/api1/201907078/call-api3", callApiHandler("API3", api3URL, 2))

	fmt.Println("Iniciando API 1 en el puerto 8081...")
	// Levantamos el servidor en el puerto 8081
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Printf("Error al iniciar el servidor: %s\n", err)
	}
}