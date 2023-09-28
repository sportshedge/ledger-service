#!/bin/bash
export "$(grep -v '^#' .env | xargs)"
# Kill process if already running on port
lsof -i:$APP_PORT -Fp | head -n 1 | sed 's/^p//' | xargs kill
lsof -i:$GRPC_PORT -Fp | head -n 1 | sed 's/^p//' | xargs kill

# Start using nodemon
 export APP_ENV=local && make ledger && nodemon --exec go run cmd/combined/main.go --signal SIGTERM

# Start without nodemon
#export APP_ENV=local && go run cmd/http/main.go --signal SIGTERM

