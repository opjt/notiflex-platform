package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/valkey-io/valkey-go"
)

const version = "v0.7.0"
const notificationsTopic = "notifications"

var valkeyClient valkey.Client
var kafkaProducer sarama.SyncProducer

func main() {
	addr := os.Getenv("VALKEY_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("VALKEY_PASSWORD")
	if pwFile := os.Getenv("VALKEY_PASSWORD_FILE"); pwFile != "" {
		if data, err := os.ReadFile(pwFile); err == nil {
			password = string(data)
		}
	}

	var err error
	for i := 0; i < 10; i++ {
		valkeyClient, err = valkey.NewClient(valkey.ClientOption{
			InitAddress: []string{addr},
			Password:    password,
		})
		if err == nil {
			break
		}
		log.Printf("Valkey 연결 재시도 %d/10: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Valkey 연결 실패: %v", err)
	}
	defer valkeyClient.Close()

	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	kafkaCfg := sarama.NewConfig()
	kafkaCfg.Version = sarama.V4_3_0_0
	kafkaCfg.Producer.Return.Successes = true

	for i := 0; i < 10; i++ {
		kafkaProducer, err = sarama.NewSyncProducer([]string{kafkaBroker}, kafkaCfg)
		if err == nil {
			break
		}
		log.Printf("Kafka Producer 연결 재시도 %d/10: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("Kafka Producer 연결 실패: %v", err)
	}
	defer kafkaProducer.Close()

	go runConsumer(kafkaBroker, kafkaCfg)

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method=%s path=%s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": version})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method=%s path=%s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/id", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method=%s path=%s", r.Method, r.URL.Path)
		pod := os.Getenv("POD_NAME")
		if pod == "" {
			pod = "local"
		}

		result, err := valkeyClient.Do(context.Background(),
			valkeyClient.B().Incr().Key("notiflex:id").Build(),
		).AsInt64()
		if err != nil {
			http.Error(w, "Valkey error", http.StatusInternalServerError)
			return
		}

		msg, _ := json.Marshal(map[string]any{"id": result, "pod": pod})
		_, _, err = kafkaProducer.SendMessage(&sarama.ProducerMessage{
			Topic: notificationsTopic,
			Value: sarama.ByteEncoder(msg),
		})
		if err != nil {
			log.Printf("Kafka 메시지 전송 실패: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":  result,
			"pod": pod,
		})
	})

	fmt.Println("Notiflex API listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func runConsumer(broker string, cfg *sarama.Config) {
	pod := os.Getenv("POD_NAME")
	if pod == "" {
		pod = "local"
	}

	var group sarama.ConsumerGroup
	var err error
	for i := 0; i < 10; i++ {
		group, err = sarama.NewConsumerGroup([]string{broker}, "notiflex-api-consumer", cfg)
		if err == nil {
			break
		}
		log.Printf("Kafka Consumer 연결 재시도 %d/10: %v", i+1, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Printf("Kafka Consumer 연결 실패: %v", err)
		return
	}
	defer group.Close()

	handler := &notificationsHandler{pod: pod}
	ctx := context.Background()
	for {
		if err := group.Consume(ctx, []string{notificationsTopic}, handler); err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				return
			}
			log.Printf("Kafka Consumer 에러: %v", err)
			time.Sleep(3 * time.Second)
		}
	}
}

type notificationsHandler struct {
	pod string
}

func (h *notificationsHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *notificationsHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *notificationsHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Printf("kafka consumed: topic=%s partition=%d offset=%d value=%s consumer_pod=%s",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Value), h.pod)
		sess.MarkMessage(msg, "")
	}
	return nil
}
