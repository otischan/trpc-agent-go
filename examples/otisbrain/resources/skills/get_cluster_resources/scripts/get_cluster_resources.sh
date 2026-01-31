#!/bin/bash

# Script to get cluster resources based on parameters
NAMESPACE=${1:-"all"}
RESOURCE_TYPE=${2:-"all"}

# Function to detect and return the appropriate command (kubectl or oc)
get_k8s_command() {
  if command -v oc >/dev/null 2>&1; then
    echo "oc"
  elif command -v kubectl >/dev/null 2>&1; then
    echo "kubectl"
  else
    echo ""
  fi
}

# Get the appropriate command to use
CMD=$(get_k8s_command)

if [ -z "$CMD" ]; then
  echo "Neither kubectl nor oc command found. Please install kubectl or oc CLI tool."
  exit 1
fi

if [ "$RESOURCE_TYPE" = "cpu" ] || [ "$RESOURCE_TYPE" = "memory" ]; then
  if [ "$NAMESPACE" = "all" ]; then
    ${CMD} top nodes 2>/dev/null || echo "${CMD} top nodes not available - ensure metrics server is installed"
  else
    ${CMD} top pods -n "$NAMESPACE" 2>/dev/null || echo "${CMD} top pods not available - ensure metrics server is installed"
  fi
elif [ "$RESOURCE_TYPE" = "pods" ]; then
  if [ "$NAMESPACE" = "all" ]; then
    ${CMD} get pods --all-namespaces -o wide 2>/dev/null || echo "${CMD} get pods failed"
  else
    ${CMD} get pods -n "$NAMESPACE" -o wide 2>/dev/null || echo "${CMD} get pods failed"
  fi
elif [ "$RESOURCE_TYPE" = "deployments" ]; then
  if [ "$NAMESPACE" = "all" ]; then
    ${CMD} get deployments --all-namespaces -o wide 2>/dev/null || echo "${CMD} get deployments failed"
  else
    ${CMD} get deployments -n "$NAMESPACE" -o wide 2>/dev/null || echo "${CMD} get deployments failed"
  fi
elif [ "$RESOURCE_TYPE" = "all" ]; then
  if [ "$NAMESPACE" = "all" ]; then
    ${CMD} get all --all-namespaces 2>/dev/null || echo "${CMD} get all failed"
  else
    ${CMD} get all -n "$NAMESPACE" 2>/dev/null || echo "${CMD} get all failed"
  fi
else
  ${CMD} get "$RESOURCE_TYPE" -n "$NAMESPACE" -o wide 2>/dev/null || echo "${CMD} get failed - unsupported resource type or insufficient permissions"
fi