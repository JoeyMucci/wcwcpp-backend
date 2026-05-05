#!/bin/bash

# Usage: bash devscripts/subcontests.sh [action]
# Uses env vars: HEADER, CONTEST_SLUG, SUBCONTEST_TITLE, JOIN_CODE, SUBCONTEST_SLUG

ACTION=$1

if [ "$ACTION" == "create" ]; then
  curl -X POST http://localhost:8080/api.v1.ContestService/CreateSubcontest \
    -H "Content-Type: application/json" \
    -H "$HEADER" \
    -d '{
      "contestSlug": "'"${CONTEST_SLUG:-my-awesome-contest}"'",
      "subcontestTitle": "'"${SUBCONTEST_TITLE:-My Test Subcontest}"'",
      "selfJoin": true
    }'
  echo ""
elif [ "$ACTION" == "list" ]; then
  curl -X POST http://localhost:8080/api.v1.ContestService/ListSubcontests \
    -H "Content-Type: application/json" \
    -H "$HEADER" \
    -d '{
      "contestSlug": "'"${CONTEST_SLUG:-my-awesome-contest}"'"
    }'
  echo ""
elif [ "$ACTION" == "join" ]; then
  curl -X POST http://localhost:8080/api.v1.ContestService/JoinSubcontest \
    -H "Content-Type: application/json" \
    -H "$HEADER" \
    -d '{
      "joinCode": "'"$JOIN_CODE"'"
    }'
  echo ""
elif [ "$ACTION" == "delete" ]; then
  curl -X POST http://localhost:8080/api.v1.ContestService/DeleteSubcontest \
    -H "Content-Type: application/json" \
    -H "$HEADER" \
    -d '{
      "subcontestSlug": "'"${SUBCONTEST_SLUG:-my-test-subcontest}"'"
    }'
  echo ""
else
  echo "Usage: bash devscripts/subcontests.sh [create|list|join|delete]"
fi
