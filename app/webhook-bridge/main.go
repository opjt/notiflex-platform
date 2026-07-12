package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type Alert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type Payload struct {
	Alerts []Alert `json:"alerts"`
}

func main() {
	torchiURL := os.Getenv("TORCHI_URL")
	if torchiURL == "" {
		log.Fatal("TORCHI_URL 환경변수가 필요합니다")
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var payload Payload
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		for _, alert := range payload.Alerts {
			msg := format(alert)
			if err := send(torchiURL, msg); err != nil {
				log.Printf("전송 실패: %v", err)
				continue
			}
			log.Printf("알림 전송: %s", strings.SplitN(msg, "\n", 2)[0])
		}

		w.WriteHeader(http.StatusOK)
	})

	log.Println("webhook-bridge listening on :8080")
	http.ListenAndServe(":8080", nil)
}

func format(a Alert) string {
	icon := "🚨"
	if a.Status == "resolved" {
		icon = "✅"
	}

	summary := a.Annotations["summary"]
	if summary == "" {
		summary = a.Labels["alertname"]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s [%s] %s", icon, a.Labels["namespace"], summary)
	if pod := a.Labels["pod"]; pod != "" {
		fmt.Fprintf(&b, "\nPod: %s", pod)
	}
	return b.String()
}

func send(url, msg string) error {
	resp, err := http.Post(url, "text/plain", bytes.NewBufferString(msg))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
