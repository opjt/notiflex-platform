# Notiflex Platform

## 프로젝트 개요

Notiflex — B2B 알림 SaaS 플랫폼

## 기술 스택

- **언어**: Go 표준 라이브러리 (외부 프레임워크 없음)
- **컨테이너**: scratch 베이스 이미지
- **인프라**: GKE Standard (Zonal), Spot VM

## GCP 설정

- **프로젝트 ID**: opjt-gitaiops-project
- **리전**: asia-southeast1 (싱가포르)
- **존**: asia-southeast1-a

## Artifact Registry

- **이미지 경로**: `asia-southeast1-docker.pkg.dev/opjt-gitaiops-project/notiflex`

## 행동 규칙

1. 항상 현재 상태를 확인한 후 실행한다
2. 변경 전 현재 상태를 먼저 확인한다
3. kubectl 명령은 반드시 `--context` 플래그를 지정한다
4. 매니패스트 작성 시 네임스페이스(`notiflex`)를 명시한다.
