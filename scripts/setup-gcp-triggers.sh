#!/usr/bin/env bash
# ==============================================================================
# Jynko — Multi-Environment Cloud Build Triggers & Cloud Run IAP Provisioner
# ==============================================================================
# Configures three branch-routed Cloud Build triggers:
#   1. dev : dev + all feature branches (--branch-pattern="^(main|stg)$" --invert-regex) -> jynko-plan-dev
#   2. stg : ^stg$                                                                       -> jynko-plan-stg
#   3. prd : ^main$                                                                      -> jynko-plan-prd
#
# Usage:
#   PROJECT_ID=aparaturafiscala-178410 ./scripts/setup-gcp-triggers.sh
#
# Optional overrides:
#   REGION=europe-west3
#   SERVICE_NAME=jynko-plan
#   RUNNER_SA=jynko-plan-runner
#   BUILDER_SA=jynko-plan-build
#   CSR_REPO=jynko-plan
#   ATTACHMENTS_BUCKET=
#   REQUIRE_PRD_APPROVAL=false
#   IAP_SETTINGS_FILE=           # e.g. deploy/iap-settings.yaml (applied to existing Cloud Run services)
# ==============================================================================
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || true)}"
if [ -z "${PROJECT_ID}" ]; then
  echo "ERROR: PROJECT_ID is not set and gcloud has no active project." >&2
  exit 1
fi

REGION="${REGION:-europe-west3}"
SERVICE_NAME="${SERVICE_NAME:-jynko-plan}"
RUNNER_SA="${RUNNER_SA:-jynko-plan-runner}"
BUILDER_SA="${BUILDER_SA:-jynko-plan-build}"
CSR_REPO="${CSR_REPO:-jynko-plan}"
ATTACHMENTS_BUCKET="${ATTACHMENTS_BUCKET:-}"
REQUIRE_PRD_APPROVAL="${REQUIRE_PRD_APPROVAL:-false}"
IAP_SETTINGS_FILE="${IAP_SETTINGS_FILE:-}"

PROJECT_NUM="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
BUILDER_SA_RESOURCE="projects/${PROJECT_ID}/serviceAccounts/${BUILDER_SA}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "==> Project: ${PROJECT_ID} (#${PROJECT_NUM}), Region: ${REGION}"

upsert_csr_trigger() {
  local env_name="$1"
  local branch_pattern="$2"
  local invert_flag="$3"
  local approval_flag="$4"

  local trigger_name="${SERVICE_NAME}-${env_name}"
  local target_service="${SERVICE_NAME}-${env_name}"
  local iap_audience="/projects/${PROJECT_NUM}/locations/${REGION}/services/${target_service}"
  local subs="_ENV_PREFIX=${env_name},_REGION=${REGION},_SERVICE_NAME=${SERVICE_NAME},_SERVICE_ACCOUNT=${RUNNER_SA},_IAP_AUDIENCE=${iap_audience}"

  if [ -n "${ATTACHMENTS_BUCKET}" ]; then
    subs="${subs},_ATTACHMENTS_GCS_BUCKET=${ATTACHMENTS_BUCKET}"
  fi

  if gcloud builds triggers describe "${trigger_name}" --project="${PROJECT_ID}" --region="${REGION}" >/dev/null 2>&1; then
    echo "==> Trigger '${trigger_name}' already exists in ${REGION}; skipping creation (delete first to recreate)."
    return 0
  fi

  echo "==> Creating Cloud Source Repository trigger '${trigger_name}' -> ${target_service}..."
  local cmd=(
    gcloud builds triggers create cloud-source-repositories
    "--project=${PROJECT_ID}"
    "--region=${REGION}"
    "--name=${trigger_name}"
    "--repo=${CSR_REPO}"
    "--branch-pattern=${branch_pattern}"
    "--build-config=cloudbuild.yaml"
    "--service-account=${BUILDER_SA_RESOURCE}"
    "--substitutions=${subs}"
  )
  if [ "${invert_flag}" = "true" ]; then
    cmd+=("--invert-regex")
  fi
  if [ "${approval_flag}" = "true" ]; then
    cmd+=("--require-approval")
  fi

  "${cmd[@]}"
}

# 1. dev: dev branch + all feature branches (anything except main and stg)
upsert_csr_trigger "dev" "^(main|stg)$" "true" "false"

# 2. stg: stg branch only (pre-release validation)
upsert_csr_trigger "stg" "^stg$" "false" "false"

# 3. prd: main branch only (production)
upsert_csr_trigger "prd" "^main$" "false" "${REQUIRE_PRD_APPROVAL}"

# Configure IAP Service Agent invoker binding (and optional iap-settings.yaml) on any deployed services
for env_name in dev stg prd; do
  svc="${SERVICE_NAME}-${env_name}"
  if gcloud run services describe "${svc}" --project="${PROJECT_ID}" --region="${REGION}" >/dev/null 2>&1; then
    echo "==> Ensuring IAP Service Agent invoker role on Cloud Run service '${svc}'..."
    gcloud run services add-iam-policy-binding "${svc}" \
      --project="${PROJECT_ID}" \
      --region="${REGION}" \
      --member="serviceAccount:service-${PROJECT_NUM}@gcp-sa-iap.iam.gserviceaccount.com" \
      --role="roles/run.invoker" >/dev/null

    if [ -n "${IAP_SETTINGS_FILE}" ] && [ -f "${IAP_SETTINGS_FILE}" ]; then
      echo "==> Applying IAP OAuth & programmaticClients settings (${IAP_SETTINGS_FILE}) to '${svc}'..."
      gcloud iap settings set "${IAP_SETTINGS_FILE}" \
        --project="${PROJECT_ID}" \
        --resource-type=cloud-run \
        --region="${REGION}" \
        --service="${svc}"
    fi
  else
    echo "    (Cloud Run service '${svc}' not deployed yet; IAP invoker binding will apply after first build.)"
  fi
done

echo "==> Multi-environment triggers ready (${SERVICE_NAME}-dev, ${SERVICE_NAME}-stg, ${SERVICE_NAME}-prd)."
