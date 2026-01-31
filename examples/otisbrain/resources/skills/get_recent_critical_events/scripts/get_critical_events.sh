#!/bin/bash

# Script to get critical events based on parameters
TIME_RANGE=${1:-"last_hour"}
SEVERITY=${2:-"all"}
RESOURCE_TYPE=${3:-"all"}

case "$TIME_RANGE" in
  "last_hour")
    hours=1
    ;;
  "last_6_hours")
    hours=6
    ;;
  "last_24_hours")
    hours=24
    ;;
  *)
    hours=1  # default to last hour
    ;;
esac

# Find the latest critical summary file using find command to ensure proper path resolution
# This finds files in the logs/critical_record directory relative to current working directory
# and sorts them by modification time (newest first)
latest_summary=$(find ./logs/critical_record -name "critical_summary_*.txt" -type f -printf '%T@ %p\n' 2>/dev/null | sort -nr | head -n1 | cut -d' ' -f2-)

if [ -n "$latest_summary" ] && [ -f "$latest_summary" ]; then
  cat "$latest_summary"
else
  echo "No aggregated critical summary files found at ./logs/critical_record/critical_summary_*.txt. The log aggregation process may not have run yet. Please wait for the first aggregation cycle to complete."
fi