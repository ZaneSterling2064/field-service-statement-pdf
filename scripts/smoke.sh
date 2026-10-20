#!/bin/sh
set -eu

curl --fail-with-body --request POST http://localhost:8080/statements \
  --header 'Content-Type: application/json' \
  --data @examples/completed-statement.json
