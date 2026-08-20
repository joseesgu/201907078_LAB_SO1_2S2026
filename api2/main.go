package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const CARNET = "201907078"
const VM_NUMBER = 1
const API_NAME = "API2"

// ... (Pega aquí las mismas estructuras HealthResponse y CallResponse del código de API1) ...
type HealthResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	VM        int    `json:"VM"`
	Carnet    string `json:"carnet"`
}

type CallResponse struct {
	Apiname    string `json:"apiname"`
	Message    string `json:"message"`
	Connection bool   `json:"connection"`
	Carnet     string `json:"carnet"`
}

// ... (Pega aquí la misma función healthHandler y callApiHandler del código de API1) ...
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := HealthResponse{
		Status:    "UP",
		Message:   fmt.Sprintf("%s is Ready", API_NAME),
		Timestamp: time.Now().Format(time.RFC3339),
		VM:        VM_NUMBER,
		Carnet:    CARNET,
	}
	json.NewEncoder(w).Encode(resp)
}

func callApiHandler(targetApi string, targetURL string, targetVM int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, err := http.Get(targetURL + "/health")
		
		if err == nil && resp.StatusCode == http.StatusOK {
			var healthResp HealthResponse
			body, _ := io.ReadAll(resp.Body)
			json.Unmarshal(body, &healthResp)
			resp.Body.Close()

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

		errorResp := CallResponse{
			Apiname:    targetApi,
			Message:    fmt.Sprintf("ERROR: The %s located on the VM%d is not working", targetApi, targetVM),
			Connection: false,
			Carnet:     CARNET,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResp)
	}
}

func main() {
	// CONFIGURA AQUÍ TUS IPs
	api1URL := "http://localhost:8081"         // API1 está en la misma VM1
	api3URL := "http://192.168.122.224:8083"

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api2/201907078/call-api1", callApiHandler("API1", api1URL, 1))
	http.HandleFunc("/api2/201907078/call-api3", callApiHandler("API3", api3URL, 2))

	fmt.Println("Iniciando API2 en puerto 8082...")
	http.ListenAndServe(":8082", nil)
}