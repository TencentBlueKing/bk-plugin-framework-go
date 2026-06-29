#!/usr/bin/env bash
set -euo pipefail

echo "[Sync] BEGIN ====================="

{{cookiecutter.project_name}} syncapigw
{{cookiecutter.project_name}} fetch-apigw-public-key

echo "[Sync] DONE ====================="
