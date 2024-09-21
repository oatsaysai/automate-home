#!/bin/sh

docker run --rm \
    --platform linux/amd64 \
    -e CMD=serve \
    -p 8090:8080 \
    --name automate-home \
    image-registry.fintblock.com/automate-home
