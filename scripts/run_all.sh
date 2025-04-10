#!/bin/sh

run() {
    echo "Starting server $1"
    sleep 1
    go run cmd/server/main.go -mp 50052 -p 5004"$1" > /tmp/log"$1".txt 2>&1
}

for i in 1 2 3 4 5 6 7 8 9 10; do
    run $i &
done

go run cmd/server/main.go

