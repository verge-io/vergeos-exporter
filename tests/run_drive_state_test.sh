#!/bin/bash

# Script to test the drive state monitoring functionality

# Default values
EXPORTER_URL="http://localhost:9888"
METRICS_PATH="/metrics"
VERGE_URL=""
VERGE_USER=""
VERGE_PASS=""
START_EXPORTER=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --exporter-url=*)
      EXPORTER_URL="${1#*=}"
      shift
      ;;
    --metrics-path=*)
      METRICS_PATH="${1#*=}"
      shift
      ;;
    --verge-url=*)
      VERGE_URL="${1#*=}"
      shift
      ;;
    --verge-user=*)
      VERGE_USER="${1#*=}"
      shift
      ;;
    --verge-pass=*)
      VERGE_PASS="${1#*=}"
      shift
      ;;
    --start-exporter)
      START_EXPORTER=true
      shift
      ;;
    --help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --exporter-url=URL     URL of the VergeOS exporter (default: http://localhost:9888)"
      echo "  --metrics-path=PATH    Path to metrics endpoint (default: /metrics)"
      echo "  --verge-url=URL        VergeOS API URL (required)"
      echo "  --verge-user=USER      VergeOS API username (required)"
      echo "  --verge-pass=PASS      VergeOS API password (required)"
      echo "  --start-exporter       Start the exporter as part of the test"
      echo "  --help                 Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      echo "Use --help for usage information"
      exit 1
      ;;
  esac
done

# Check required parameters
if [ -z "$VERGE_URL" ] || [ -z "$VERGE_USER" ] || [ -z "$VERGE_PASS" ]; then
  echo "Error: VergeOS URL, username, and password are required"
  echo "Use --help for usage information"
  exit 1
fi

# Build the test program
echo "Building drive state test program..."
cd "$(dirname "$0")"
go build -o drive_state_test drive_state_test.go

# Run the test
echo "Running drive state test..."
./drive_state_test \
  --exporter.url="$EXPORTER_URL" \
  --exporter.metrics-path="$METRICS_PATH" \
  --verge.url="$VERGE_URL" \
  --verge.username="$VERGE_USER" \
  --verge.password="$VERGE_PASS" \
  --start-exporter="$START_EXPORTER"

# Clean up
rm -f drive_state_test