# Notiflex 여정 기록

이 파일은 독자가 실제로 진행한 내용을 기록한다. AI가 각 챕터 완료 시 자동으로 업데이트한다.

## 진행 현황

| 챕터 | 서브챕터 | 상태 | 완료일 | 비고 |
|------|---------|------|--------|------|
| ch2 | 2.2 설치 확인 | ✅ | 2026-06-29 | statusline 건너뜀 |
| ch2 | 2.3 gcloud 설정 | ✅ | 2026-06-29 | 싱가포르 리전 선택 |
| ch2 | 2.4 GitHub 저장소 | ✅ | 2026-06-29 | public 저장소로 생성 |
| ch2 | 2.5 GKE 클러스터 | ✅ | 2026-06-29 | asia-southeast1-a |
| ch2 | 2.6 빌드/배포 | ✅ | 2026-06-29 | Cloud Build 사용 |
| ch2 | 2.7 첫 커밋 | ✅ | 2026-06-29 | |
| ch3 | 3.2 GitOps 도구 | ✅ | 2026-07-07 | ArgoCD 선택, automated sync+selfHeal |
| ch3 | 3.3 기능 추가 | ✅ | 2026-07-07 | /version 엔드포인트 추가, Rolling Update 확인 |
| ch3 | 3.4 CI | ✅ | 2026-07-07 | GitHub Actions, GCP_SA_KEY 방식 |
| ch3 | 3.5 CI-CD 연결 | ✅ | 2026-07-07 | CI→deployment.yaml 자동 업데이트→ArgoCD 자동 Sync |
| ch4 | 4.2 메트릭 모니터링 | ✅ | 2026-07-12 | kube-prometheus-stack, Prometheus+Grafana |
| ch4 | 4.3 로그 수집 | ✅ | 2026-07-12 | Loki+Fluent Bit, SingleBinary 모드 |
| ch4 | 4.4 알림 | ✅ | 2026-07-12 | PrometheusRule+Alertmanager+webhook-bridge→torchi.app |
| ch5 | 5.2 트래픽 관리 | ⬜ | | |
| ch5 | 5.3 무중단 배포 | ⬜ | | |
| ch6 | 6.1 캐시 | ⬜ | | |
| ch6 | 6.2 시크릿 관리 | ⬜ | | |
| ch6 | 6.3 Canary 전환 | ⬜ | | |
| ch7 | 7.2 멀티 노드풀 | ⬜ | | |
| ch7 | 7.3 App of Apps | ⬜ | | |
| ch7 | 7.4 멀티테넌시 | ⬜ | | |
| ch8 | 8.1 메시징 | ⬜ | | |
| ch8 | 8.2 트레이싱 | ⬜ | | |
| ch8 | 8.3 CronJob | ⬜ | | |
| ch9 | 9.1 저장소 분석 | ⬜ | | |
| ch9 | 9.2 회고 | ⬜ | | |
| ch9 | 9.3 온보딩 문서 | ⬜ | | |
| ch9 | 9.4 GitAIOps 분석 | ⬜ | | |
| ch9 | 9.5 마무리 | ⬜ | | |

## 도구 선택 기록

| 영역 | 선택 | 검토한 대안 | 선택 이유 |
|------|------|-----------|----------|
| 리전 | asia-southeast1 (싱가포르) | us-central1, asia-northeast3 | 비용과 지연 균형 |
| 이미지 빌드 | Cloud Build | 로컬 Docker | Docker Desktop 없이 빌드 가능, CI와 동일한 방식 |
| GitHub 저장소 | public | private | 실습 편의성 |
| GitOps 도구 (ADR-001) | ArgoCD | Flux | CRD 기반 Application 리소스, UI 제공, 3분 폴링 자동 Sync |
| CI 도구 (ADR-002) | GitHub Actions | Cloud Build, Jenkins | 저장소와 동일 플랫폼, secrets 연동 간편, 무료 |
| 메트릭 모니터링 (ADR-003) | Prometheus + Grafana | Datadog, Victoria Metrics | CNCF 졸업 프로젝트, kube-prometheus-stack으로 한번에 설치, PromQL 표준 |
| 로그 수집 (ADR-004) | Loki + Fluent Bit | ELK, CloudWatch | 라벨 기반 인덱싱으로 저비용, Grafana와 통합, DaemonSet 경량 수집 |
| 알림 연동 (ADR-005) | PrometheusRule + Alertmanager + webhook-bridge | PagerDuty, OpsGenie | GitOps 호환 CRD, 커스텀 Push 서비스(torchi.app) 연동 위해 브릿지 직접 구현 |

## 현재 버전

| 컴포넌트 | 버전 | 변경 이력 |
|---------|------|----------|
| Go | 1.25 | |
| Notiflex 이미지 | sha-a6d73fc | HTTP 요청 로그 추가 (ch4.3) |
| ArgoCD | v2.x (latest stable) | ch3.2에서 설치 |
| Prometheus | v3.13.1 | kube-prometheus-stack (ch4.2) |
| Grafana | 13.1.0 | kube-prometheus-stack (ch4.2) |
| Loki | 3.6.7 | SingleBinary 모드 (ch4.3) |
| Fluent Bit | 2.1.0 | grafana/fluent-bit-plugin-loki (ch4.3) |
| webhook-bridge | v0.1.0 | torchi.app 알림 브릿지 (ch4.4) |
| Kafka | | |
| OTel SDK | | |

## 현재 리소스

| 노드풀 | 머신 타입 | 노드 수 | 주요 워크로드 |
|--------|----------|---------|-------------|
| default-pool | e2-medium | 2 (Spot VM) | notiflex-api |

## 트러블슈팅 이력

| 챕터 | 문제 | 해결 |
|------|------|------|
| ch2.3 | opjt-gitaiops-project 프로젝트 없음 | 새로 생성 |
| ch2.6 | Cloud Build API 미활성화 | cloudbuild.googleapis.com, storage.googleapis.com 활성화 |
| ch2.6 | Cloud Build → Artifact Registry 권한 없음 | roles/artifactregistry.writer 부여 |
| ch2.6 | GKE 노드 → Artifact Registry 권한 없음 | roles/artifactregistry.reader 부여 |
| ch3.2 | ArgoCD CRD 설치 실패 | --server-side=true --force-conflicts=true 옵션 사용 |
| ch3.2 | ArgoCD NetworkPolicy가 GitHub 접근 차단 | kubectl delete networkpolicy -n argocd --all |
| ch3.4 | GitHub Actions manifest push 403 | gh api로 default_workflow_permissions=write 설정 |
| ch3.4 | can_approve_pull_request_reviews 타입 오류 | gh api -f 대신 -F 플래그 사용 |
| ch4.2 | ArgoCD 비밀번호 리셋 실패 | htpasswd -nbBC 10으로 올바른 bcrypt 해시 생성 후 argocd-secret 패치 |
| ch4.3 | Loki chunks/results cache Pending | CPU/메모리 부족 → chunksCache/resultsCache enabled: false |
| ch4.3 | Fluent Bit deprecated chart hardcoded URL | ConfigMap 직접 패치로 loki-gateway 주소 변경 (helm upgrade 시 초기화 주의) |
| ch4.3 | Fluent Bit mem buf overlimit | requests 64Mi → 128Mi, limits 256Mi 증가 |
| ch4.4 | webhook-bridge x509 인증서 에러 | scratch 이미지에 CA 인증서 없음 → alpine builder에서 ca-certificates.crt 복사 |
| ch4.4 | PodRestarting 알림 rollout restart로 미트리거 | rollout restart는 Pod 교체라 restarts 카운트 미증가 → kubectl debug + busybox로 kill 1 |
