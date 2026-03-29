#!/bin/bash
set -e

echo "Starting Ollama server..."
ollama serve &

echo "Waiting for Ollama to be ready..."
for i in $(seq 1 60); do
  if curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "Ollama is ready"
    break
  fi
  sleep 2
done

echo "Pulling tinyllama model..."
ollama pull tinyllama

echo "Starting auth proxy..."
exec /usr/local/bin/proxy
