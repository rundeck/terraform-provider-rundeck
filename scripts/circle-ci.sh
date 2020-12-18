#!/bin/bash

set -euo pipefail

export RUNDECK_AUTH_TOKEN=1d08bf61-f962-467f-8ba3-ab8a463b3467
export RUNDECK_URL=http://127.0.0.1:4440

wait_for_rd() (
    while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' localhost:4440)" != "302" ]]; do sleep 1; done
)

export -f wait_for_rd

(
    cd test
    docker-compose up --build -d
)

timeout 60s bash -c wait_for_rd

go clean --testcache
make testacc