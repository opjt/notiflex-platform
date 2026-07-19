# Notiflex Platform — 아키텍처 스냅샷

> **이 파일의 목적**: AI가 매 대화에서 전체 그림을 빠르게 파악할 수 있도록 현재 시점의 아키텍처 상태를 기록한다.
> 마지막 업데이트: ch6.4 (2026-07-19)

## 3층 지식 구조

| 문서 | 역할 | 자동 로드 |
|------|------|-----------|
| `CLAUDE.md` | AI에게 프로젝트 메타데이터·가드레일 제공 | 매 대화 자동 로드 |
| `claude-context/` | 현재 아키텍처 스냅샷 (이 파일) | 필요 시 참조 |
| `docs/architecture-decisions.md` | ADR — 결정 이유의 누적 기록 | 사람·AI 함께 검토 |

셋을 분리하는 이유: 메모리(작업 컨텍스트)·아키텍처(현재 그림)·결정(과거 누적)이 섞이면 AI가 맥락을 오독한다.

---

## 클러스터 토폴로지

| 항목 | 값 |
|------|-----|
| 클러스터 이름 | `notiflex-cluster` |
| GCP 프로젝트 | `opjt-gitaiops-project` |
| 리전 / 존 | `asia-southeast1` (싱가포르) |
| 노드풀 | `default-pool` — e2-standard-2 × 2 (Spot VM) |
| Kubernetes 버전 | v1.35.5-gke.1241004 |
| 컨테이너 런타임 | containerd 2.1.7 |
| 활성화된 GKE 기능 | Gateway API, Workload Identity, Secrets Store CSI, GKE Managed Prometheus |
| 노드 내부 IP | `10.148.15.193`, `10.148.0.60` |
| 노드 외부 IP | `34.87.96.208`, `34.87.159.4` |

---

## 컴포넌트 다이어그램

```
[인터넷]
    │  HTTP :80
    ▼
[GCP L7 Regional External LB]  34.2.146.184
    │  GKE Gateway API (gke-l7-regional-external-managed)
    ▼
[Gateway: notiflex-gateway]  namespace: notiflex
    │  HTTPRoute: notiflex-route  (PathPrefix: /)
    ▼
[Service: notiflex-api]  ClusterIP 34.118.238.210:80
    │
    ▼
[Argo Rollout: notiflex-api]  strategy: Canary (20%→50%→80%→100%, 각 30s pause)
    ├── stable  ReplicaSet: notiflex-api-594485b98f  (현재 운영)
    │       └── Pod: notiflex-api-594485b98f-xkg26  (v0.6.0, node: yw96)
    └── canary  ReplicaSet: (배포 시 생성)
    │
    │  ServiceAccount: notiflex-api
    │    annotation: iam.gke.io/gcp-service-account
    │      → notiflex-api@opjt-gitaiops-project.iam.gserviceaccount.com
    │
    ├── [Secrets Store CSI]  driver: secrets-store-gke.csi.k8s.io
    │       SecretProviderClass: notiflex-secrets
    │         → GCP Secret Manager: valkey-password/versions/latest
    │       Mount: /mnt/secrets/valkey-password (read-only)
    │
    └── [Valkey]  StatefulSet: valkey-primary  (Helm: valkey-6.2.0 / app: 9.1.0)
            Service: valkey-primary  ClusterIP 34.118.238.78:6379
            env: VALKEY_ADDR=valkey-primary.notiflex.svc.cluster.local:6379
            env: VALKEY_PASSWORD_FILE=/mnt/secrets/valkey-password
            용도: INCR 카운터 (notiflex:id)

[Service: notiflex-api-preview]  ClusterIP 34.118.238.131:80
    └── Canary 배포 중 새 버전 트래픽 수신 (canaryService)
```

---

## 배포 파이프라인

```
[개발자 git push]
    │
    ▼
[GitHub Actions CI]  .github/workflows/ci.yaml
    ├── go build (Dockerfile: golang:1.25-alpine → scratch)
    ├── docker push → Artifact Registry
    │     asia-southeast1-docker.pkg.dev/opjt-gitaiops-project/notiflex/api:<sha>
    └── k8s/smb/rollout.yaml 이미지 태그 업데이트 후 커밋 push
    │
    ▼
[ArgoCD]  namespace: argocd
    Application: notiflex-smb  (Synced / Healthy)
    대상: k8s/smb/  디렉터리
    sync: auto (selfHeal: true)  ← Git이 유일한 변경 통로
    │
    ▼
[Argo Rollouts]  namespace: argo-rollouts
    Canary 전략으로 점진적 트래픽 이동 후 stable 승격
```

---

## 관측 가능성

| 도구 | Helm 차트 | 버전 | 역할 |
|------|-----------|------|------|
| Prometheus | kube-prometheus-stack | 87.15.1 (app: v0.92.1) | 메트릭 수집·저장 |
| Grafana | kube-prometheus-stack 포함 | — | 메트릭 시각화 대시보드 |
| Alertmanager | kube-prometheus-stack 포함 | — | 알림 라우팅 |
| Loki | loki-7.0.0 | app: 3.6.7 | 로그 수집·저장 |
| Fluent Bit | fluent-bit-2.6.0 | app: v2.1.0 | Pod 로그 → Loki 전송 |
| webhook-bridge | 커스텀 Deployment | — | Alertmanager → 외부 웹훅 변환 |
| PrometheusRule | notiflex-alerts | — | 커스텀 알림 규칙 |

모두 `monitoring` 네임스페이스에 위치. Grafana 접속: `kubectl port-forward svc/kube-prometheus-grafana 3000:80 -n monitoring`

---

## 주요 네임스페이스

| 네임스페이스 | 주요 워크로드 | 비고 |
|-------------|--------------|------|
| `notiflex` | Rollout(notiflex-api), StatefulSet(valkey-primary), Service×3, SecretProviderClass | 앱 운영 영역 |
| `argocd` | argocd-server, argocd-application-controller, argocd-repo-server 등 6종 | GitOps 컨트롤 플레인 |
| `argo-rollouts` | argo-rollouts controller | Canary/Blue-Green 배포 엔진 |
| `monitoring` | Prometheus, Grafana, Alertmanager, Loki, Fluent Bit, webhook-bridge | 관측 가능성 스택 |
| `kube-system` | CSI Secrets Store, kube-proxy, kube-dns, fluentbit-gke 등 | GKE 시스템 |

---

## 현재 앱 상태

| 항목 | 값 |
|------|-----|
| 앱 버전 | v0.6.0 |
| 이미지 태그 | `sha-6a8de73` |
| 배포 전략 | Canary (20%→50%→80%→100%) |
| Gateway 외부 IP | `34.2.146.184` |
| 엔드포인트 | `GET /health`, `GET /version`, `GET /id` |
| Valkey 용도 | `/id` 요청 시 INCR 원자 카운터 |

---

## Workload Identity 흐름 (시크릿 접근)

```
Pod (serviceAccountName: notiflex-api)
  └─► K8s SA annotation → GCP SA (notiflex-api@opjt-gitaiops-project.iam.gserviceaccount.com)
        └─► GKE OIDC 토큰 → GCP STS 검증
              └─► Secret Manager IAM (roles/secretmanager.secretAccessor)
                    └─► valkey-password → CSI 드라이버 → /mnt/secrets/valkey-password
```

JSON 키 파일 없음. GKE 클러스터 자체가 OIDC 발급자 역할.
