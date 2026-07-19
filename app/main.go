package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/valkey-io/valkey-go"
)

const version = "v0.6.0"

var valkeyClient valkey.Client

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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":  result,
			"pod": pod,
		})
	})

	fmt.Println("Notiflex API listening on :8080")
	http.ListenAndServe(":8080", nil)
}
