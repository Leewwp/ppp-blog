#!/bin/bash

# Halo鍗氬鑷姩閮ㄧ讲鑴氭湰
# 鐢ㄩ€旓細浠嶨itHub鎷夊彇鏈€鏂颁唬鐮佸苟鑷姩閮ㄧ讲

set -e

# 閰嶇疆鍙橀噺
PROJECT_DIR="/opt/halo-blog"
LOG_FILE="/var/log/halo-deploy.log"
BACKUP_DIR="/opt/halo-blog/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# 棰滆壊杈撳嚭
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 鏃ュ織鍑芥暟
log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a $LOG_FILE
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a $LOG_FILE
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a $LOG_FILE
}

# 妫€鏌ユ槸鍚︿互root鐢ㄦ埛杩愯
if [ "$EUID" -ne 0 ]; then
    error "璇蜂娇鐢╮oot鐢ㄦ埛杩愯姝よ剼鏈?
    exit 1
fi

log "=========================================="
log "寮€濮嬮儴缃?Halo 鍗氬"
log "=========================================="

# 鍒涘缓澶囦唤鐩綍
mkdir -p $BACKUP_DIR

# 澶囦唤褰撳墠閰嶇疆
log "澶囦唤褰撳墠閰嶇疆..."
cp $PROJECT_DIR/docker-compose.yml $BACKUP_DIR/docker-compose.yml.$DATE.backup
cp $PROJECT_DIR/.env $BACKUP_DIR/.env.$DATE.backup 2>/dev/null || true

# 杩涘叆椤圭洰鐩綍
cd $PROJECT_DIR

# 妫€鏌ユ槸鍚︿负Git浠撳簱
if [ ! -d ".git" ]; then
    error "褰撳墠鐩綍涓嶆槸Git浠撳簱锛岃鍏堝垵濮嬪寲Git浠撳簱"
    exit 1
fi

# 鎷夊彇鏈€鏂颁唬鐮?log "鎷夊彇鏈€鏂颁唬鐮?.."
git fetch origin
git checkout main
git pull --ff-only origin main

# 妫€鏌ocker-compose.yml鏄惁瀛樺湪
if [ ! -f "docker-compose.yml" ]; then
    error "docker-compose.yml 鏂囦欢涓嶅瓨鍦?
    exit 1
fi

# 鍋滄褰撳墠鏈嶅姟
log "鍋滄褰撳墠鏈嶅姟..."

# 鎷夊彇鏈€鏂伴暅鍍?log "鎷夊彇鏈€鏂癉ocker闀滃儚..."
docker-compose pull || true

# 鍚姩鏈嶅姟
log "鍚姩鏈嶅姟..."
docker-compose up -d --remove-orphans

# 绛夊緟鏈嶅姟鍚姩
log "绛夊緟鏈嶅姟鍚姩..."
sleep 10

# 妫€鏌ユ湇鍔＄姸鎬?log "妫€鏌ユ湇鍔＄姸鎬?.."
docker-compose ps

# 娓呯悊鏈娇鐢ㄧ殑闀滃儚
log "娓呯悊鏈娇鐢ㄧ殑Docker闀滃儚..."
docker image prune -f

# 娓呯悊鏈娇鐢ㄧ殑鍗凤紙鍙€夛級
# docker volume prune -f

# 鏄剧ず鏃ュ織
log "=========================================="
log "閮ㄧ讲瀹屾垚锛?
log "=========================================="
log "鏌ョ湅鏈嶅姟鏃ュ織: docker-compose logs -f"
log "鏌ョ湅鏈嶅姟鐘舵€? docker-compose ps"
log "鏌ョ湅閮ㄧ讲鏃ュ織: tail -f $LOG_FILE"

# 鍙戦€侀€氱煡锛堝彲閫夛級
# 濡傛灉閰嶇疆浜嗛拤閽夋垨浼佷笟寰俊webhook锛屽彲浠ュ湪杩欓噷娣诲姞閫氱煡
# curl -X POST "your_webhook_url" -d "閮ㄧ讲瀹屾垚"

exit 0
