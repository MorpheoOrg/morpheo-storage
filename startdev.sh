#!/bin/bash
echo ""
echo "Compiling Go storage packages..."
rm build/target &> /dev/null
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/target .

echo ""
echo "Build and run docker containers..."
# sudo chmod 777 -R data
STORAGE_PORT=8081 STORAGE_AUTH_USER=u STORAGE_AUTH_PASSWORD=p docker-compose up --build
