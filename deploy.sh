#!/bin/bash

# Halo博客自动部署脚本
# 用途：从GitHub拉取最新代码并自动部署

set -e

# 配置变量
PROJECT_DIR="/opt/halo-blog"
LOG_FILE="/var/log/halo-deploy.log"
BACKUP_DIR="/opt/halo-blog/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a $LOG_FILE
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a $LOG_FILE
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a $LOG_FILE
}

# 检查是否以root用户运行
if [ "$EUID" -ne 0 ]; then
    error "请使用root用户运行此脚本"
    exit 1
fi

log "=========================================="
log "开始部署 Halo 博客"
log "=========================================="

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份当前配置
log "备份当前配置..."
cp $PROJECT_DIR/docker-compose.yml $BACKUP_DIR/docker-compose.yml.$DATE.backup
cp $PROJECT_DIR/.env $BACKUP_DIR/.env.$DATE.backup 2>/dev/null || true

# 进入项目目录
cd $PROJECT_DIR

# 检查是否为Git仓库
if [ ! -d ".git" ]; then
    error "当前目录不是Git仓库，请先初始化Git仓库"
    exit 1
fi

# 拉取最新代码
log "拉取最新代码..."
git fetch origin
git reset --hard origin/main

# 检查docker-compose.yml是否存在
if [ ! -f "docker-compose.yml" ]; then
    error "docker-compose.yml 文件不存在"
    exit 1
fi

# 停止当前服务
log "停止当前服务..."
docker-compose down

# 拉取最新镜像
log "拉取最新Docker镜像..."
docker-compose pull

# 启动服务
log "启动服务..."
docker-compose up -d

# 等待服务启动
log "等待服务启动..."
sleep 10

# 检查服务状态
log "检查服务状态..."
docker-compose ps

# 清理未使用的镜像
log "清理未使用的Docker镜像..."
docker image prune -f

# 清理未使用的卷（可选）
# docker volume prune -f

# 显示日志
log "=========================================="
log "部署完成！"
log "=========================================="
log "查看服务日志: docker-compose logs -f"
log "查看服务状态: docker-compose ps"
log "查看部署日志: tail -f $LOG_FILE"

# 发送通知（可选）
# 如果配置了钉钉或企业微信webhook，可以在这里添加通知
# curl -X POST "your_webhook_url" -d "部署完成"

exit 0
