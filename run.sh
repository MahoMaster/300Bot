#!/bin/bash

set -e

BASE_DIR=$(cd "$(dirname "$0")"; pwd)
PROJECT_NAME=$(basename "$BASE_DIR")
SERVICE_NAME="${PROJECT_NAME}.service"

cd $BASE_DIR

echo "clean old file"
rm -rf $PROJECT_NAME

if [ -f build ]; then
    echo "start new process"

    cp ./build ./$PROJECT_NAME
    chmod 777 $PROJECT_NAME

    echo "restart service"
    systemctl restart ${SERVICE_NAME}

    echo "service restarted"
else
    echo "build file not found"
    exit 1
fi