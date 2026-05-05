#!/bin/bash

# Ensure required commands exist
command -v curl >/dev/null 2>&1 || { echo >&2 "curl is required but it's not installed.  Aborting."; exit 1; }
command -v jq >/dev/null 2>&1 || { echo >&2 "jq is required but it's not installed.  Aborting."; exit 1; }

COMMAND=$1

if [ -z "$COMMAND" ]; then
    echo "Usage: $0 <command>"
    echo "Commands:"
    echo "  count   - Get the total number of users"
    exit 1
fi

case $COMMAND in
    count)
        curl -s -X POST http://localhost:8080/api.v1.UsersService/CountUsers \
            -H "Content-Type: application/json" \
            -d '{}' | jq .
        ;;
    delete)
        curl -s -X POST http://localhost:8080/api.v1.UsersService/DeleteUser \
            -H "Content-Type: application/json" \
            -H "$HEADER" \
            -d '{}' | jq .
        ;;
    *)
        echo "Unknown command: $COMMAND"
        exit 1
        ;;
esac
