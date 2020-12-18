#!/bin/bash

TESTARGS=$1

trap _catch ERR
function _catch {
    scripts/stop_docker.sh
    exit 1
}
scripts/start_docker.sh
RUNDECK_AUTH_TOKEN=N4n5Lw6wYlIeJlxGUUyslWWelifGAQsF RUNDECK_URL=http://localhost:4440 TF_ACC=1 go test -count=1 ./... -v ${TESTARGS} -timeout 120m
scripts/stop_docker.sh
