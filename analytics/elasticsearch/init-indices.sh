#!/bin/bash

# Wait for Elasticsearch to be ready
echo "Waiting for Elasticsearch to be ready..."
until curl -s http://localhost:9200/_cluster/health | grep -q '"status":"green\|yellow"'; do
  sleep 5
done

echo "Creating Elasticsearch indices..."

# Create events index
curl -X PUT "localhost:9200/events" -H 'Content-Type: application/json' -d @mappings/events.json

# Create servers index
curl -X PUT "localhost:9200/servers" -H 'Content-Type: application/json' -d @mappings/servers.json

# Create metrics index
curl -X PUT "localhost:9200/metrics" -H 'Content-Type: application/json' -d @mappings/metrics.json

# Create feedback index
curl -X PUT "localhost:9200/feedback" -H 'Content-Type: application/json' -d @mappings/feedback.json

echo "Indices created successfully!"