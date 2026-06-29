# Notiflex Platform

B2B 알림 SaaS 플랫폼 — 「AI 시대에 개발자가 알아야 하는 인프라 구성 배포 with 클로드 코드」 실습 저장소

## 스택

- **언어**: Go 1.25 (표준 라이브러리만 사용)
- **컨테이너**: scratch 베이스 멀티스테이지 빌드
- **인프라**: GKE Standard (asia-southeast1), Spot VM
- **이미지 레지스트리**: Artifact Registry

## API 엔드포인트

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/health` | 헬스체크 |
| GET | `/id` | 순차 ID 생성 + 응답 파드 이름 반환 |

## 디렉터리 구조

```
notiflex-platform/
├── app/
│   ├── main.go        # Go 애플리케이션
│   └── Dockerfile     # 멀티스테이지 빌드
├── k8s/
│   └── smb/           # K8s 매니페스트
├── .github/
│   └── workflows/     # CI 파이프라인
├── cloudbuild.yaml    # Cloud Build 설정
└── JOURNEY.md         # 실습 진행 기록
```
