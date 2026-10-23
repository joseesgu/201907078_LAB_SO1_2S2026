package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type WarReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

func main() {
	// 1. Conexión a Valkey (Ajusta el host si tu Service se llama distinto)
	rdb := redis.NewClient(&redis.Options{
		Addr:     "valkey-server-valkeyinstance:6379",
		Password: "",
		DB:       0,
	})
	ctx := context.Background()

	// Probar conexión a Valkey
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Fallo al conectar a Valkey: %v", err)
	}
	log.Println("Conexión a Valkey establecida exitosamente.")

	// 2. Conexión a RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-svc:5672/")
	if err != nil {
		log.Fatalf("Fallo al conectar a RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Fallo al abrir un canal: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"war_reports", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Fallo al declarar la cola: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, "", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Fallo al registrar consumidor: %v", err)
	}

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)

	// 3. Procesar y guardar en Valkey
	go func() {
		for d := range msgs {
			var report WarReport
			if err := json.Unmarshal(d.Body, &report); err != nil {
				log.Printf("Error decodificando JSON: %v", err)
				continue
			}

			// Generar un ID único basado en el país y la fecha
			key := fmt.Sprintf("report:%s:%d", report.Country, time.Now().UnixNano())

			// Guardar el JSON entero con un TTL de 1 hora para evitar saturación de memoria
			err = rdb.Set(ctx, key, string(d.Body), time.Hour).Err()
			if err != nil {
				log.Printf("Error guardando en Valkey: %v", err)
			} else {
				log.Printf("Guardado en Valkey [Key: %s] -> País: %s | Aviones: %d | Barcos: %d", 
					key, report.Country, report.WarplanesInAir, report.WarshipsInWater)
			}
		}
	}()

	log.Printf(" [*] Consumer en Go conectado a Valkey y esperando reportes...")
	<-forever
}