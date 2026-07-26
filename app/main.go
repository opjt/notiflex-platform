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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const version = "v0.8.0"
const notificationsTopic = "notifications"

var valkeyClient valkey.Client
var kafkaProducer sarama.SyncProducer
var tracer = otel.Tracer("notiflex-api")

func initTracer(ctx context.Context) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", "notiflex-api"),
	))
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

func main() {
	tracerShutdown, err := initTracer(context.Background())
	if err != nil {
		log.Printf("OTel 트레이서 초기화 실패: %v", err)
		tracerShutdown = func(context.Context) error { return nil }
	}
	defer tracerShutdown(context.Background())

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
		_, span := tracer.Start(r.Context(), "GET /version")
		defer span.End()
		log.Printf("method=%s path=%s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": version})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, span := tracer.Start(r.Context(), "GET /health")
		defer span.End()
		log.Printf("method=%s path=%s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	http.HandleFunc("/id", func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "GET /id")
		defer span.End()
		log.Printf("method=%s path=%s", r.Method, r.URL.Path)
		pod := os.Getenv("POD_NAME")
		if pod == "" {
			pod = "local"
		}

		valkeyCtx, valkeySpan := tracer.Start(ctx, "valkey.incr")
		result, err := valkeyClient.Do(valkeyCtx,
			valkeyClient.B().Incr().Key("notiflex:id").Build(),
		).AsInt64()
		valkeySpan.End()
		if err != nil {
			span.RecordError(err)
			http.Error(w, "Valkey error", http.StatusInternalServerError)
			return
		}

		_, kafkaSpan := tracer.Start(ctx, "kafka.produce")
		msg, _ := json.Marshal(map[string]any{"id": result, "pod": pod})
		_, _, err = kafkaProducer.SendMessage(&sarama.ProducerMessage{
			Topic: notificationsTopic,
			Value: sarama.ByteEncoder(msg),
		})
		kafkaSpan.End()
		if err != nil {
			span.RecordError(err)
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
