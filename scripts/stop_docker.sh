#!/bin/bash
echo "Stopping docker containers"
docker kill rundeck-test
docker rm -f rundeck-test
