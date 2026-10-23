package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type WarReport struct {
	Country         string `json:"country"`
	WarplanesInAir  int32  `json:"warplanes_in_air"`
	WarshipsInWater int32  `json:"warships_in_water"`
	Timestamp       string `json:"timestamp"`
}

// Variables globales para tracking analítico en memoria
var (
	minPlanes int32 = 999999
	maxPlanes int32 = 0
	minShips  int32 = 999999
	maxShips  int32 = 0
	planeFreq       = make(map[int32]int)
	shipFreq        = make(map[int32]int)
	mu        sync.Mutex
)

// Función para calcular la moda (el valor que más se repite)
func getMode(freq map[int32]int) int32 {
	var mode int32
	maxCount := 0
	for k, v := range freq {
		if v > maxCount {
			maxCount = v
			mode = k
		}
	}
	return mode
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "valkey-server-valkeyinstance:6379",
	})
	ctx := context.Background()

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq-svc:5672/")
	if err != nil {
		log.Fatalf("Error RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, _ := conn.Channel()
	defer ch.Close()

	q, _ := ch.QueueDeclare("war_reports", true, false, false, false, nil)
	msgs, _ := ch.Consume(q.Name, "", true, false, false, false, nil)

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for d := range msgs {
			var report WarReport
			json.Unmarshal(d.Body, &report)

			mu.Lock()
			if report.WarplanesInAir > maxPlanes { maxPlanes = report.WarplanesInAir }
			if report.WarplanesInAir < minPlanes { minPlanes = report.WarplanesInAir }
			planeFreq[report.WarplanesInAir]++
			modePlanes := getMode(planeFreq)

			if report.WarshipsInWater > maxShips { maxShips = report.WarshipsInWater }
			if report.WarshipsInWater < minShips { minShips = report.WarshipsInWater }
			shipFreq[report.WarshipsInWater]++
			modeShips := getMode(shipFreq)
			mu.Unlock()

			pipe := rdb.Pipeline()

			pipe.ZIncrBy(ctx, "top:aviones", float64(report.WarplanesInAir), report.Country)
			pipe.ZIncrBy(ctx, "top:barcos", float64(report.WarshipsInWater), report.Country)
			pipe.HSet(ctx, "stats:global", 
				"max_aviones", maxPlanes, "min_aviones", minPlanes, "moda_aviones", modePlanes,
				"max_barcos", maxShips, "min_barcos", minShips, "moda_barcos", modeShips,
			)

			country := report.Country
			if country == "gtm" || country == "GTM" {
				pipe.Incr(ctx, "stats:gtm_total") 
				pipe.XAdd(ctx, &redis.XAddArgs{
					Stream: "series:gtm",
					Values: map[string]interface{}{
						"Aviones": report.WarplanesInAir,
						"Barcos":  report.WarshipsInWater,
					},
				})
			}

			_, err := pipe.Exec(ctx)
			if err == nil {
				// ¡Fuera del if! Imprimirá a todos los países
				log.Printf("Procesado [%s] -> Aviones: %d | Barcos: %d", report.Country, report.WarplanesInAir, report.WarshipsInWater)
			}
		}
	}()

	log.Printf(" [*] Consumidor Analítico conectado a Valkey...")
	<-forever
}