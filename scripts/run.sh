#!/bin/bash

image_name="telegram-youtube-reteller"
container_name="telegram-youtube-reteller"

docker build -t "$image_name" .
docker stop "$container_name" &>/dev/null || true
docker rm "$container_name" &>/dev/null || true

docker run -d \
    --env-file .env \
    --name "$container_name" \
    "$image_name"

docker image prune -f