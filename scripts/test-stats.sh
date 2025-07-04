#!/bin/bash

# Test script for stats endpoints
BASE_URL="${1:-http://localhost:8080}"
echo "Testing stats endpoints at: $BASE_URL"
echo ""

# Test global stats
echo "1. Testing global stats endpoint..."
curl -s "$BASE_URL/vp/stats/global" | jq '.' || echo "Failed to get global stats"
echo ""

# Test leaderboard
echo "2. Testing leaderboard endpoint..."
curl -s "$BASE_URL/vp/stats/leaderboard?type=installs&limit=5" | jq '.' || echo "Failed to get leaderboard"
echo ""

# Test servers with sorting
echo "3. Testing sorted servers endpoint..."
curl -s "$BASE_URL/vp/servers?sort=rating&limit=5" | jq '.servers[].name' || echo "Failed to get sorted servers"
echo ""

# Test server stats (using a known server ID if available)
SERVER_ID="anthropic-model-context-protocol"
echo "4. Testing individual server stats for: $SERVER_ID"
curl -s "$BASE_URL/vp/servers/$SERVER_ID/stats" | jq '.' || echo "Failed to get server stats"
echo ""

# Test aggregated stats
echo "5. Testing aggregated server stats..."
curl -s "$BASE_URL/vp/servers/$SERVER_ID/stats?aggregated=true" | jq '.' || echo "Failed to get aggregated stats"
echo ""

# Test community source filter
echo "6. Testing community source filter..."
curl -s "$BASE_URL/vp/stats/global?source=COMMUNITY" | jq '.' || echo "Failed to get community stats"
echo ""

echo "Stats endpoint tests completed!"