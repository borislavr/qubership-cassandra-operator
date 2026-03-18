#!/usr/bin/env bash

#Required for DP job to obtain parameters
eval $(sed -e 's/:[^:\/\/]/="/g;s/$/"/g;s/ *=/=/g' <<<"$DEPLOYMENT_PARAMETERS" | grep "ENABLE_MIGRATION\|CUSTOM_RESOURCE_NAME")

ENABLE_MIGRATION=${ENABLE_MIGRATION:-true}
SERVICE_NAME=${SERVICE_NAME:-cassandra-operator}
SELECTOR=${SELECTOR:-"app.kubernetes.io/part-of"=cassandra}

echo "ENABLE_MIGRATION: ${ENABLE_MIGRATION}"
if [[ ${ENABLE_MIGRATION} != "true" ]]; then
  exit 0
fi

if command -v kubectl &>/dev/null; then
  kubectl="kubectl"
else
  source ${WORKSPACE}/oc_version_used.sh
  kubectl="${OCBINVERP}"
fi

if command -v helm &>/dev/null; then
  helm="helm"
else
  helm="helm3"
fi

echo "Start migration procedure"

if ! ($helm list | grep ${SERVICE_NAME}); then
  echo "There are no ${SERVICE_NAME} helm releases. Please perform manual migration"
  exit 0
fi

if ! $kubectl get cassandraservices cassandra-operator; then
  echo "Cassandraservice does not exist. No migration needed."
  exit 0
fi


echo "Removing the linkage between deployments and services to keep them running during migration"
$kubectl get deployments,services,statefulsets -l ${SELECTOR} -o name | xargs -I {} $kubectl patch {} -p '{"metadata":{"ownerReferences":null}}'


echo "End migration procedure"