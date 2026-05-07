#!/bin/bash

set -e

echo "init service..."

# 当前目录
BASE_DIR=$(cd "$(dirname "$0")"; pwd)

# 项目名 = 当前目录名
PROJECT_NAME=$(basename "$BASE_DIR")

SERVICE_NAME="${PROJECT_NAME}.service"

LOG_DIR="${BASE_DIR}/logs"

echo "project: $PROJECT_NAME"
echo "dir: $BASE_DIR"

# 创建 logs
if [ ! -d "$LOG_DIR" ]; then
    echo "create logs directory"
    mkdir -p "$LOG_DIR"
fi

# 检查 cronolog
if ! command -v cronolog >/dev/null 2>&1; then
    echo "install cronolog"
    yum install -y cronolog || apt-get install -y cronolog
fi

SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}"

echo "create systemd service: $SERVICE_FILE"

cat > $SERVICE_FILE <<EOF
[Unit]
Description=${PROJECT_NAME} Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${BASE_DIR}

ExecStart=/bin/bash -lc 'exec ${BASE_DIR}/${PROJECT_NAME} 2>&1 | /usr/sbin/cronolog ${BASE_DIR}/logs/%%Y-%%m-%%d.log'

Restart=always
RestartSec=3

KillMode=control-group
TimeoutStopSec=10
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF


echo "reload systemd"
systemctl daemon-reload

echo "enable service"
systemctl enable ${SERVICE_NAME}

echo "service status"
systemctl status ${SERVICE_NAME}

echo "init finished"