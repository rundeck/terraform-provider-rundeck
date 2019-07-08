#!/bin/bash
docker run -p 4440:4440 -v ${PWD}/props/tokens.properties:/api/tokens.properties:Z -e RUNDECK_TOKENS_FILE=/api/tokens.properties -d --name rundeck-test rundeck/rundeck:3.3.4
until $(curl --output /dev/null --silent --head --fail http://localhost:4440); do 
    printf '.' 
    sleep 2 
done 
printf '\n'
