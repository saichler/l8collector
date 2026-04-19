#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WEBHOOK_YAML="${SCRIPT_DIR}/validating-webhook.yaml"
ADMISSION_YAML="${SCRIPT_DIR}/admission-control.yaml"

kubectl delete -f "${ADMISSION_YAML}" --ignore-not-found
kubectl delete -f "${WEBHOOK_YAML}" --ignore-not-found

