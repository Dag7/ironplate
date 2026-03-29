#!/bin/bash
# =============================================================================
# Find GCP Billing Service ID
# =============================================================================
# This script finds service IDs in the GCP Cloud Billing Catalog API.
# Service IDs are in hex format (e.g., 24E6-581D-38E5) and are required
# for billing budget filters.
#
# Usage:
#   ./scripts/find-service-id.sh [search_term]
#
# Examples:
#   ./scripts/find-service-id.sh "Vertex AI"
#   ./scripts/find-service-id.sh "BigQuery"
#   ./scripts/find-service-id.sh "Cloud Run"
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Search term (default to Vertex AI)
SEARCH_TERM="${1:-Vertex AI}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}GCP Billing Service ID Finder${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Searching for: ${YELLOW}${SEARCH_TERM}${NC}"
echo ""

# Check for required tools
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is required but not installed.${NC}"
    exit 1
fi

if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}Error: gcloud CLI is required but not installed.${NC}"
    exit 1
fi

# Get access token
echo -e "${BLUE}[1/2]${NC} Authenticating with GCP..."
ACCESS_TOKEN=$(gcloud auth print-access-token 2>/dev/null)
if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}Error: Failed to get access token. Run 'gcloud auth login' first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Authenticated"

# Search through pages
echo -e "${BLUE}[2/2]${NC} Searching Cloud Billing Catalog API..."
echo ""

TOKEN=""
PAGE=0
FOUND=0
RESULTS=""

while true; do
    PAGE=$((PAGE + 1))

    # Build URL
    if [ -z "$TOKEN" ]; then
        URL="https://cloudbilling.googleapis.com/v1/services?pageSize=500"
    else
        URL="https://cloudbilling.googleapis.com/v1/services?pageSize=500&pageToken=$TOKEN"
    fi

    # Fetch page
    echo -ne "\r${YELLOW}Scanning page ${PAGE}...${NC}                    "
    RESPONSE=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" "$URL")

    # Check for errors
    ERROR=$(echo "$RESPONSE" | jq -r '.error.message // empty')
    if [ -n "$ERROR" ]; then
        echo ""
        echo -e "${RED}API Error: $ERROR${NC}"
        exit 1
    fi

    # Search for matches in this page
    MATCHES=$(echo "$RESPONSE" | jq -r --arg search "$SEARCH_TERM" \
        '.services[] | select(.displayName | test($search; "i")) | "\(.serviceId)\t\(.displayName)"')

    if [ -n "$MATCHES" ]; then
        RESULTS="${RESULTS}${MATCHES}\n"
        FOUND=$((FOUND + $(echo "$MATCHES" | wc -l)))
    fi

    # Get next page token
    TOKEN=$(echo "$RESPONSE" | jq -r '.nextPageToken // empty')

    # Exit if no more pages
    if [ -z "$TOKEN" ]; then
        break
    fi
done

echo -e "\r${GREEN}✓${NC} Scanned ${PAGE} pages                    "
echo ""

# Display results
if [ $FOUND -eq 0 ]; then
    echo -e "${YELLOW}No services found matching '${SEARCH_TERM}'${NC}"
    echo ""
    echo "Try a different search term, e.g.:"
    echo "  ./scripts/find-service-id.sh 'AI Platform'"
    echo "  ./scripts/find-service-id.sh 'Machine Learning'"
else
    echo -e "${GREEN}Found ${FOUND} matching service(s):${NC}"
    echo ""
    echo -e "${BLUE}SERVICE ID\t\tDISPLAY NAME${NC}"
    echo "----------------------------------------"
    echo -e "$RESULTS" | sort -t$'\t' -k2 | while IFS=$'\t' read -r id name; do
        if [ -n "$id" ]; then
            echo -e "${GREEN}${id}${NC}\t${name}"
        fi
    done
    echo ""
    echo -e "${BLUE}Usage in Pulumi budget filter:${NC}"
    FIRST_ID=$(echo -e "$RESULTS" | head -1 | cut -f1)
    echo "  services: [\"services/${FIRST_ID}\"]"
fi

echo ""
