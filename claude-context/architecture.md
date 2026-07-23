# Notiflex Platform — 아키텍처 스냅샷

> **이 파일의 목적**: AI가 매 대화에서 전체 그림을 빠르게 파악할 수 있도록 현재 시점의 아키텍처 상태를 기록한다.
> 마지막 업데이트: ch7.4 (2026-07-23)

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
| 노드풀 | `default-pool`(e2-standard-2 ×2, Spot), `api-pool`(e2-medium ×1, Spot), `worker-pool`(e2-standard-2 ×1, Spot) — `ops-pool`은 리전 외부 IP 쿼터(IN_USE_ADDRESSES 4/4) 초과로 미생성 |
| Kubernetes 버전 | v1.35.5-gke.1241004 |
| 컨테이너 런타임 | containerd 2.1.7 |
| 활성화된 GKE 기능 | Gateway API, Workload Identity, Secrets Store CSI, GKE Managed Prometheus |
| 워크로드 배치 | `notiflex-api`(smb, enterprise) → `api-pool` (nodeSelector: `cloud.google.com/gke-nodepool: api-pool`), Valkey·모니터링 스택 → `default-pool`, `worker-pool`은 아직 워크로드 없음(ch8.1 Kafka 예정) |

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
[Argo Rollout: notiflex-api]  namespace: notiflex, replicas: 2, strategy: Canary (20%→50%→80%→100%, 각 30s pause)
    │  nodeSelector: cloud.google.com/gke-nodepool: api-pool
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
    └── [Valkey]  StatefulSet: valkey-primary  (Helm: valkey-6.2.0 / app: 9.1.0, default-pool)
            Service: valkey-primary  ClusterIP, port 6379
            env: VALKEY_ADDR=valkey-primary.notiflex.svc.cluster.local:6379
            env: VALKEY_PASSWORD_FILE=/mnt/secrets/valkey-password
            용도: INCR 카운터 (notiflex:id) — smb·enterprise 테넌트가 공유

[Service: notiflex-api-preview]  namespace: notiflex
    └── Canary 배포 중 새 버전 트래픽 수신 (canaryService)

--- 멀티테넌시: enterprise 테넌트 (ch7.4) ---

[Argo Rollout: notiflex-api]  namespace: enterprise, replicas: 1, strategy: Canary
    │  nodeSelector: cloud.google.com/gke-nodepool: api-pool (smb와 같은 노드풀 공유)
    │  ServiceAccount: notiflex-api (enterprise ns 전용, 동일 GCP SA에 Workload Identity 바인딩 추가)
    │
    ├── [Secrets Store CSI]  SecretProviderClass: notiflex-secrets (enterprise ns)
    │       → GCP Secret Manager: valkey-password/versions/latest  (smb와 동일 secret 재사용)
    │
    └── cross-namespace DNS로 notiflex ns의 Valkey에 연결
            VALKEY_ADDR=valkey-primary.notiflex.svc.cluster.local:6379
            ⚠ Namespace 분리는 논리적 격리일 뿐, NetworkPolicy 없어 네트워크상 두 테넌트 상호 접근 가능
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
    Application: root-app  (Synced / Healthy) — App of Apps 루트 (ch7.3)
      source: argocd/apps/  (directory.recurse: true)
      │
      ├── Application: notiflex-smb  (Synced / Healthy, sync-wave: 2)
      │     대상: k8s/smb/  디렉터리 → namespace: notiflex
      │
      └── Application: notiflex-enterprise  (Synced / Healthy, sync-wave: 2)
            대상: k8s/enterprise/  디렉터리 → namespace: enterprise (CreateNamespace=true)

    sync: auto (selfHeal: true)  ← Git이 유일한 변경 통로
    │
    ▼
[Argo Rollouts]  namespace: argo-rollouts
    Canary 전략으로 점진적 트래픽 이동 후 stable 승격 (smb·enterprise 각각 독립 Rollout)
```

> ⚠️ Prometheus/Grafana/Loki/Fluent Bit/Valkey는 Helm으로 직접 설치되어 ArgoCD Application으로 관리되지 않는다. App of Apps가 관리하는 건 `notiflex-smb`, `notiflex-enterprise` 두 개뿐이다.

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
| `notiflex` | Rollout(notiflex-api, replicas:2), StatefulSet(valkey-primary), Service×3, SecretProviderClass | smb 테넌트 운영 영역 |
| `enterprise` | Rollout(notiflex-api, replicas:1), Service×2, SecretProviderClass, ServiceAccount | enterprise 테넌트 (ch7.4 신설), Valkey는 notiflex ns 공유 |
| `argocd` | argocd-server, argocd-application-controller, argocd-repo-server 등 6종 + root-app/notiflex-smb/notiflex-enterprise Application | GitOps 컨트롤 플레인, App of Apps 루트 |
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
| Gateway 외부 IP | `34.2.146.184` (notiflex 테넌트, enterprise는 Gateway 미연결) |
| 엔드포인트 | `GET /health`, `GET /version`, `GET /id` |
| Valkey 용도 | `/id` 요청 시 INCR 원자 카운터, smb·enterprise 공유 |
| notiflex replicas | 2 (ch7.2에서 1→2로 확장) |
| enterprise replicas | 1 |
| 노드 배치 | smb·enterprise 모두 `api-pool` (nodeSelector) |

---

## Workload Identity 흐름 (시크릿 접근)

```
Pod (namespace: notiflex,   serviceAccountName: notiflex-api)
Pod (namespace: enterprise, serviceAccountName: notiflex-api)  ← ch7.4에서 추가 바인딩
  └─► K8s SA annotation → GCP SA (notiflex-api@opjt-gitaiops-project.iam.gserviceaccount.com)
        └─► GKE OIDC 토큰 → GCP STS 검증
              └─► Secret Manager IAM (roles/secretmanager.secretAccessor)
                    └─► valkey-password → CSI 드라이버 → /mnt/secrets/valkey-password
```

GCP SA 하나(`notiflex-api@...`)에 두 네임스페이스(`notiflex`, `enterprise`)의 K8s SA가 모두 `workloadIdentityUser`로 바인딩되어 있다 (`svc.id.goog[notiflex/notiflex-api]`, `svc.id.goog[enterprise/notiflex-api]`). JSON 키 파일 없음. GKE 클러스터 자체가 OIDC 발급자 역할.
