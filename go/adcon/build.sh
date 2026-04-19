#!/usr/bin/env bash
set -e
docker build --no-cache --platform=linux/amd64 -t saichler/probler-admission:latest .
docker push saichler/probler-admission:latest
