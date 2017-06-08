#!/bin/bash

# Pass argument "0" to not compile Go packages and use the previous compilation.
BUILD=$1
if [ -z $BUILD ]; then
    BUILD=1
fi

if [ "$BUILD" -ne "0" ]; then
    echo ""
    echo "Compiling Go storage packages..."
    rm build/target &> /dev/null
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/target .
fi


echo ""
echo "Build and run docker containers..."
# sudo chmod 777 -R data
STORAGE_PORT=8081 STORAGE_AUTH_USER=u STORAGE_AUTH_PASSWORD=p docker-compose up --build
