package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "wartweets-proto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WarReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func getCountryEnum(country string) pb.Countries {
	switch country {
	case "USA": return pb.Countries_usa
	case "RUS": return pb.Countries_rus
	case "CHN": return pb.Countries_chn
	case "ESP": return pb.Countries_esp
	case "GTM": return pb.Countries_gtm
	default:    return pb.Countries_countries_unknown
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	var report WarReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Apunta al servicio interno de Kubernetes que crearemos para el gRPC Server
	conn, err := grpc.Dial("go-server-svc:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		http.Error(w, "Error conectando a gRPC", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	client := pb.NewWarReportServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	req := &pb.WarReportRequest{
		Country:         getCountryEnum(report.Country),
		WarplanesInAir:  report.WarplanesInAir,
		WarshipsInWater: report.WarshipsInWater,
		Timestamp:       report.Timestamp,
	}

	res, err := client.SendReport(ctx, req)
	if err != nil {
		log.Printf("Error gRPC: %v", err)
		http.Error(w, "Error enviando reporte", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": res.Status})
}

func main() {
	http.HandleFunc("/send", handleRequest)
	fmt.Println("Go Client escuchando en el puerto 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}