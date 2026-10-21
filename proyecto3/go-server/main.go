package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	pb "wartweets-proto/proto"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWarReportServiceServer
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

// Intercepta la llamada gRPC SendReport
func (s *server) SendReport(ctx context.Context, req *pb.WarReportRequest) (*pb.WarReportResponse, error) {
	log.Printf("Recibido vía gRPC - País: %v, Aviones: %d, Barcos: %d", req.GetCountry(), req.GetWarplanesInAir(), req.GetWarshipsInWater())

	// Conexión a RabbitMQ usando el nombre del Service de Kubernetes
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-svc:5672/")
	if err != nil {
		log.Printf("Fallo al conectar a RabbitMQ: %v", err)
		return &pb.WarReportResponse{Status: "Error conectando a RabbitMQ"}, err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return &pb.WarReportResponse{Status: "Error abriendo canal RabbitMQ"}, err
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"war_reports", // nombre de la cola
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return &pb.WarReportResponse{Status: "Error declarando cola"}, err
	}

	// Transformar los datos a JSON para encolarlos
	body, _ := json.Marshal(map[string]interface{}{
		"country":           req.GetCountry().String(),
		"warplanes_in_air":  req.GetWarplanesInAir(),
		"warships_in_water": req.GetWarshipsInWater(),
		"timestamp":         req.GetTimestamp(),
	})

	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		return &pb.WarReportResponse{Status: "Error publicando en RabbitMQ"}, err
	}

	return &pb.WarReportResponse{Status: "Reporte encolado en RabbitMQ exitosamente"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Fallo al escuchar en el puerto 50051: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterWarReportServiceServer(s, &server{})

	fmt.Println("Servidor gRPC escuchando en el puerto 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo al servir: %v", err)
	}
}