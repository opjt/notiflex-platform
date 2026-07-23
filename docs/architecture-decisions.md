# Architecture Decision Records

## ADR-001: GitOps 도구로 ArgoCD 선택 (3장)
**시점**: 2026-07 / **결정**: GitOps 도구로 ArgoCD를 채택하고 Flux는 사용하지 않는다
**이유**:
- CRD 기반 Application 리소스로 GitOps 상태를 선언적으로 정의 가능
- 웹 UI에서 배포 상태, diff, 히스토리를 시각적으로 확인 가능
- 3분 폴링 + automated sync + selfHeal으로 git 상태와 클러스터를 자동 일치
- ArgoCD 생태계(Rollouts, ApplicationSet)로 이후 장에서 점진적 확장 가능

## ADR-002: CI 도구로 GitHub Actions 선택 (3장)
**시점**: 2026-07 / **결정**: CI 도구로 GitHub Actions를 채택하고 Cloud Build, Jenkins는 사용하지 않는다
**이유**:
- 소스 저장소와 동일 플랫폼이라 별도 연동 설정 없이 push 이벤트 트리거 가능
- GCP_SA_KEY secrets 연동이 간단하고 marketplace actions 재사용 가능
- 공개 저장소 기준 무료, 별도 인프라 운영 불필요
- YAML 선언으로 GitOps 방식과 일관된 파이프라인 관리

## ADR-003: 메트릭 모니터링으로 Prometheus + Grafana 선택 (4장)
**시점**: 2026-07 / **결정**: 메트릭 모니터링으로 Prometheus + Grafana를 채택하고 Datadog, Victoria Metrics는 사용하지 않는다
**이유**:
- CNCF 졸업 프로젝트로 K8s 생태계 표준, 방대한 exporter 생태계 보유
- kube-prometheus-stack Helm 차트로 alertmanager, node-exporter까지 한번에 설치
- PromQL이 K8s 메트릭 쿼리 표준으로 자료와 예제가 풍부
- 오픈소스라 Datadog 대비 라이선스 비용 없음

## ADR-004: 로그 수집으로 Loki + Fluent Bit 선택 (4장)
**시점**: 2026-07 / **결정**: 로그 수집으로 Loki + Fluent Bit을 채택하고 ELK Stack, CloudWatch는 사용하지 않는다
**이유**:
- Loki는 라벨 기반 인덱싱으로 Elasticsearch 대비 스토리지 비용이 낮음
- Grafana와 동일 생태계라 메트릭·로그를 하나의 대시보드에서 조회 가능
- Fluent Bit은 DaemonSet으로 각 노드에서 경량 수집, Fluentd 대비 메모리 사용 낮음
- SingleBinary 모드로 소규모 클러스터에서 간단하게 운영 가능

## ADR-005: 알림 연동으로 PrometheusRule + Alertmanager + webhook-bridge 선택 (4장)
**시점**: 2026-07 / **결정**: 알림 연동으로 PrometheusRule + Alertmanager를 채택하고 PagerDuty, OpsGenie는 사용하지 않는다
**이유**:
- PrometheusRule이 CRD 기반이라 GitOps 방식으로 알림 규칙 선언적 관리 가능
- Alertmanager가 kube-prometheus-stack에 포함되어 추가 설치 불필요
- torchi.app 커스텀 Push 서비스 연동을 위해 webhook-bridge를 직접 구현하여 유연성 확보
- 외부 유료 서비스(PagerDuty) 없이 자체 알림 파이프라인 구성

## ADR-006: 외부 트래픽 진입점으로 GKE Gateway API 선택 (5장)
**시점**: 2026-07 / **결정**: 외부 트래픽 진입점으로 GKE Gateway API(gke-l7-regional-external-managed)를 채택하고 NGINX Ingress, Istio Gateway는 사용하지 않는다
**이유**:
- GKE 네이티브 관리형 L7 LB로 별도 Ingress Controller 운영 불필요
- HealthCheckPolicy CRD로 헬스체크 경로(/health:8080)를 선언적으로 설정 가능
- Gateway API가 Ingress를 대체하는 K8s 표준으로 장기적 방향성과 일치
- HTTPRoute로 트래픽 라우팅 규칙을 세분화하여 이후 Canary 전략 확장 용이

## ADR-007: 무중단 배포 전략으로 Argo Rollouts Blue/Green 선택 (5장)
**시점**: 2026-07 / **결정**: 무중단 배포 전략으로 Argo Rollouts Blue/Green을 채택하고 Flagger, K8s Rolling Update는 사용하지 않는다
**이유**:
- ArgoCD와 동일한 Argo 생태계라 ArgoCD UI에서 Rollout 상태를 통합 확인 가능
- CRD 선언으로 배포 전략을 GitOps 방식으로 관리, Deployment와 동일한 인터페이스
- Green 준비 완료 후 트래픽 전환이라 Rolling Update 대비 사용자 영향 최소화
- 5장 Blue/Green → 6장 Canary로 전략 전환 시 Rollout spec 수정만으로 가능

## ADR-008: 분산 캐시로 Valkey 선택 (6장)
**시점**: 2026-07 / **결정**: Pod 간 공유 카운터에 Valkey를 채택, Redis 및 Memcached 미사용
**이유**:
- Redis의 SSPL 라이선스 변경(2018) 이후 BSD 라이선스를 유지하는 공식 포크로 기업 환경에서 법적 리스크 없음
- Redis 프로토콜 완전 호환이라 클라이언트 교체 없이 `valkey-go` 라이브러리 사용 가능
- INCR 명령어로 원자적 카운터 보장 — 멀티 Pod 환경에서 중복 ID 없이 분산 상태 공유
- Memcached 대비 영속성(AOF/RDB)과 데이터 구조 다양성 제공, 향후 세션·큐 등으로 확장 가능

## ADR-009: 시크릿 관리로 Secrets Store CSI + GCP Secret Manager + Workload Identity 선택 (6장)
**시점**: 2026-07 / **결정**: CSI 드라이버 파일 마운트 방식 채택, External Secrets Operator 및 환경변수 직접 주입 미사용
**이유**:
- Workload Identity로 JSON 키 파일 없이 GKE OIDC 토큰만으로 GCP SA 인증 — 자격증명이 클러스터에 저장되지 않음
- 시크릿이 K8s Secret 오브젝트로 생성되지 않아 `kubectl get secret`으로 평문 노출 불가
- SecretProviderClass CRD로 어떤 시크릿을 어디에 마운트할지 Git으로 선언적 관리
- GCP Secret Manager의 버전 관리·감사 로그·자동 교체 기능을 그대로 활용

## ADR-010: 배포 전략을 Blue/Green에서 Canary로 전환 (6장)
**시점**: 2026-07 / **결정**: Argo Rollouts strategy를 blueGreen → canary(20%→50%→80%→100%)로 변경, 새 도구 도입 없음
**이유**:
- 전체 사용자가 아닌 일부(20%)에게 먼저 노출해 장애 영향 범위를 최소화
- Blue/Green 대비 리소스 효율 개선 (2x → 1.2x), 소규모 클러스터에서 CPU 여유 확보
- 각 단계(30초)에서 메트릭·로그 관찰 후 abort 가능, 자동 승격까지 안전 관찰 시간 확보
- 동일한 Rollout CRD에서 strategy 필드만 교체하여 새 컨트롤러 없이 전략 진화

## ADR-011: 워크로드 노드 배치로 nodeSelector + 멀티 노드풀 선택 (7장)
**시점**: 2026-07 / **결정**: 역할별 노드풀(api-pool, worker-pool)을 생성하고 nodeSelector로 배치, taint/toleration 및 nodeAffinity는 사용하지 않는다
**이유**:
- GKE가 노드풀 생성 시 `cloud.google.com/gke-nodepool` 라벨을 자동 부여해 커스텀 라벨 관리가 불필요
- YAML에 nodeSelector 한 줄만 추가하면 되는 가장 단순한 방식
- taint/toleration 대비 학습 곡선이 낮고, 이 규모의 클러스터에서 "거부" 기능까지는 불필요
- 단일 존 클러스터라 topology spread 등 AZ 분산 전략은 의미가 없음
- ops-pool은 리전 외부 IP 쿼터(IN_USE_ADDRESSES 4/4) 초과로 생성 보류, api-pool·worker-pool 2개로 우선 구성

## ADR-012: 여러 ArgoCD 앱 관리로 App of Apps 패턴 선택 (7장)
**시점**: 2026-07 / **결정**: root-app이 `argocd/apps/` 디렉터리를 감시하는 App of Apps 구조를 채택, ApplicationSet 및 수동 관리는 사용하지 않는다
**이유**:
- 관리할 Application이 5~7개 수준으로, ApplicationSet의 템플릿 기능 없이도 순수 YAML로 충분
- `argocd/apps/`에 파일만 추가하면 root-app이 자동으로 인식해 배포, kubectl apply 누락 위험 제거
- Git 디렉터리 자체가 클러스터 상태의 단일 진실 공급원이 되어 GitOps 원칙에 충실
- 기존 `argocd/notiflex-smb.yaml`을 디렉터리만 이동해 다운타임 없이 root-app 관리로 전환 가능

## ADR-013: 멀티테넌시로 Namespace 분리 + per-tenant Rollout 선택 (7장)
**시점**: 2026-07 / **결정**: 테넌트(enterprise)별로 별도 Namespace와 Rollout을 두는 구조를 채택, 단일 namespace+라벨 격리 및 vCluster는 사용하지 않는다
**이유**:
- e2-medium 노드 소수 구성의 단일 클러스터에서 vCluster·클러스터 분리는 비용·운영 복잡도 대비 실익이 없음
- Namespace + RBAC만으로 추가 도구 없이 논리적 격리와 리소스 그룹화가 가능
- 방금 도입한 App of Apps 구조(`argocd/apps/`)에 테넌트 Application을 그대로 추가할 수 있어 관리 패턴이 일관됨
- cross-namespace DNS로 공유 Valkey에 접근하게 하여, 클러스터 확장 없이 테넌트 간 공유 리소스 접근 패턴을 학습

## ADR-014: enterprise 시크릿 관리를 GCP Secret Manager + CSI로 통일 (7장)
**시점**: 2026-07 / **결정**: enterprise 테넌트의 Valkey 비밀번호도 smb와 동일하게 Secret Manager + CSI + Workload Identity 방식으로 전환하고, 평문 K8s Secret은 사용하지 않는다
**이유**:
- 저장소가 public이라 K8s Secret(base64, 사실상 평문)을 커밋하면 실제 비밀번호가 그대로 노출됨
- smb와 enterprise가 동일한 Valkey 인스턴스를 공유하므로 Secret Manager의 시크릿도 하나만 두면 충분
- 기존 GCP SA(`notiflex-api@...`)에 enterprise KSA용 Workload Identity 바인딩만 추가하면 되어 신규 IAM 자원 없이 확장 가능
- smb와 동일한 CSI 마운트 패턴을 재사용해 테넌트가 늘어나도 시크릿 관리 방식이 하나로 통일됨
