#!/usr/bin/env bash
# ==============================================================================
# Jynko — Multi-Environment Cloud Build Triggers & Cloud Run IAP Provisioner
# ==============================================================================
# Configures strictly three branch-routed Cloud Build triggers (no triggers for
# feature branches), with exact trigger & Cloud Run service names:
#   1. dev : ^dev$   -> trigger: jynko-plan-dev  -> Cloud Run: jynko-plan-dev
#   2. stg : ^stg$   -> trigger: jynko-plan-stg  -> Cloud Run: jynko-plan-stg
#   3. prd : ^main$  -> trigger: jynko-plan-prd  -> Cloud Run: jynko-plan-prd
#
# Usage:
#   PROJECT_ID=aparaturafiscala-178410 ./scripts/setup-gcp-triggers.sh
#
# Optional overrides:
#   REGION=europe-west3
#   RUNNER_SA=jynko-plan-runner
#   BUILDER_SA=jynko-plan-build
#   CSR_REPO=jynko-plan
#   ATTACHMENTS_BUCKET=
#   REQUIRE_PRD_APPROVAL=false
#   RECREATE_TRIGGERS=false      # set to true to replace existing jynko-plan-{dev,stg,prd} triggers
#   IAP_SETTINGS_FILE=           # e.g. iap-settings.yaml (applied to deployed Cloud Run services)
# ==============================================================================
set -euo pipefail

# Canonical base name (intentionally NOT reading $SERVICE_NAME from shell env so
# legacy exports like SERVICE_NAME=jynko-plan-app never produce jynko-plan-app-*).
readonly SERVICE_BASE="jynko-plan"

PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || true)}"
if [ -z "${PROJECT_ID}" ]; then
  echo "ERROR: PROJECT_ID is not set and gcloud has no active project." >&2
  exit 1
fi

REGION="${REGION:-europe-west3}"
RUNNER_SA="${RUNNER_SA:-jynko-plan-runner}"
BUILDER_SA="${BUILDER_SA:-jynko-plan-build}"
CSR_REPO="${CSR_REPO:-jynko-plan}"
ATTACHMENTS_BUCKET="${ATTACHMENTS_BUCKET:-}"
REQUIRE_PRD_APPROVAL="${REQUIRE_PRD_APPROVAL:-false}"
RECREATE_TRIGGERS="${RECREATE_TRIGGERS:-false}"
IAP_SETTINGS_FILE="${IAP_SETTINGS_FILE:-}"

PROJECT_NUM="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
BUILDER_SA_RESOURCE="projects/${PROJECT_ID}/serviceAccounts/${BUILDER_SA}@${PROJECT_ID}.iam.gserviceaccount.com"

echo "==> Project: ${PROJECT_ID} (#${PROJECT_NUM}), Region: ${REGION}, Base: ${SERVICE_BASE}"

# Clean up any legacy misnamed triggers (jynko-plan-app-*) if present
for env_name in dev stg prd; do
  legacy_trigger="jynko-plan-app-${env_name}"
  if gcloud builds triggers describe "${legacy_trigger}" --project="${PROJECT_ID}" --region="${REGION}" >/dev/null 2>&1; then
    echo "==> Deleting misnamed trigger '${legacy_trigger}' in ${REGION}..."
    gcloud builds triggers delete "${legacy_trigger}" --project="${PROJECT_ID}" --region="${REGION}" --quiet
  fi
done

upsert_csr_trigger() {
  local env_name="$1"
  local branch_pattern="$2"
  local approval_flag="$3"

  local trigger_name="${SERVICE_BASE}-${env_name}"
  local target_service="${SERVICE_BASE}-${env_name}"
  local iap_audience="/projects/${PROJECT_NUM}/locations/${REGION}/services/${target_service}"
  local subs="_ENV_PREFIX=${env_name},_REGION=${REGION},_SERVICE_NAME=${SERVICE_BASE},_SERVICE_ACCOUNT=${RUNNER_SA},_IAP_AUDIENCE=${iap_audience},_ATTACHMENTS_GCS_PREFIX=${env_name}"

  if [ -n "${ATTACHMENTS_BUCKET}" ]; then
    subs="${subs},_ATTACHMENTS_GCS_BUCKET=${ATTACHMENTS_BUCKET}"
  fi

  if gcloud builds triggers describe "${trigger_name}" --project="${PROJECT_ID}" --region="${REGION}" >/dev/null 2>&1; then
    if [ "${RECREATE_TRIGGERS}" = "true" ]; then
      echo "==> Recreating existing trigger '${trigger_name}' in ${REGION}..."
      gcloud builds triggers delete "${trigger_name}" --project="${PROJECT_ID}" --region="${REGION}" --quiet
    else
      echo "==> Trigger '${trigger_name}' already exists in ${REGION}; skipping (set RECREATE_TRIGGERS=true to replace)."
      return 0
    fi
  fi

  echo "==> Creating Cloud Source Repository trigger '${trigger_name}' (${branch_pattern}) -> Cloud Run '${target_service}'..."
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
  if [ "${approval_flag}" = "true" ]; then
    cmd+=("--require-approval")
  fi

  "${cmd[@]}"
}

# Strictly 3 branch triggers (no build triggers for feature branches):
# 1. dev: dev branch -> jynko-plan-dev
upsert_csr_trigger "dev" "^dev$" "false"

# 2. stg: stg branch -> jynko-plan-stg
upsert_csr_trigger "stg" "^stg$" "false"

# 3. prd: main branch -> jynko-plan-prd
upsert_csr_trigger "prd" "^main$" "${REQUIRE_PRD_APPROVAL}"

# Configure IAP Service Agent invoker binding (and optional iap-settings.yaml) on any deployed services
for env_name in dev stg prd; do
  svc="${SERVICE_BASE}-${env_name}"
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

echo "==> Done! Active triggers & target Cloud Run services:"
echo "    - jynko-plan-dev (branch: ^dev$)"
echo "    - jynko-plan-stg (branch: ^stg$)"
echo "    - jynko-plan-prd (branch: ^main$)"
