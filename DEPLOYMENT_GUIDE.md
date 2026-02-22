# Halo 博客服务器部署完整指南

本指南将帮助您完成从本地开发环境到生产环境的完整部署，包括服务器配置、域名解析和HTTPS证书配置。

## 目录

1. [前置准备](#前置准备)
2. [服务器配置](#服务器配置)
3. [域名解析](#域名解析)
4. [HTTPS证书配置](#https证书配置)
5. [Nginx反向代理配置](#nginx反向代理配置)
6. [安全加固](#安全加固)
7. [备份策略](#备份策略)
8. [监控和维护](#监控和维护)

---

## 前置准备

### 1. 所需资源

- **云服务器**：推荐配置
  - CPU: 2核及以上
  - 内存: 4GB及以上
  - 硬盘: 40GB及以上
  - 带宽: 3Mbps及以上
  - 操作系统: Ubuntu 20.04/22.04 或 CentOS 7/8

- **域名**：已购买的域名（如 example.com）

- **SSL证书**：Let's Encrypt免费证书或付费证书

### 2. 服务器环境要求

```bash
# 检查系统版本
cat /etc/os-release

# 检查Docker版本（需要24+）
docker --version

# 检查Docker Compose版本（需要2.x+）
docker-compose --version

# 检查Git版本（需要2.30+）
git --version
```

---

## 服务器配置

### 步骤1：连接服务器

```bash
# 使用SSH连接服务器
ssh root@your_server_ip

# 或使用密钥连接
ssh -i /path/to/your/key.pem root@your_server_ip
```

### 步骤2：更新系统

```bash
# Ubuntu/Debian
apt update && apt upgrade -y

# CentOS/RHEL
yum update -y
```

### 步骤3：安装Docker

```bash
# 安装Docker
curl -fsSL https://get.docker.com | sh

# 启动Docker服务
systemctl start docker
systemctl enable docker

# 验证安装
docker --version
```

### 步骤4：安装Docker Compose

```bash
# 下载Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose

# 添加执行权限
chmod +x /usr/local/bin/docker-compose

# 验证安装
docker-compose --version
```

### 步骤5：创建项目目录

```bash
# 创建项目目录
mkdir -p /opt/halo-blog
cd /opt/halo-blog

# 创建数据目录
mkdir -p halo-data mysql-data redis-data
```

### 步骤6：上传项目文件

**方法一：使用SCP上传**

```bash
# 在本地执行
scp -r /path/to/ppp-blog root@your_server_ip:/opt/halo-blog/
```

**方法二：使用Git克隆**

```bash
# 在服务器上执行
cd /opt/halo-blog
git clone https://github.com/halo-dev/halo.git
```

**方法三：直接创建docker-compose.yml**

```bash
# 在服务器上创建docker-compose.yml
nano /opt/halo-blog/docker-compose.yml
```

### 步骤7：修改docker-compose.yml

根据服务器环境调整配置：

```yaml
services:
  halo:
    image: halohub/halo:2.22
    container_name: halo
    restart: always
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      halo-network:
    ports:
      - "127.0.0.1:8090:8090"  # 只监听本地，通过Nginx代理
    volumes:
      - ./halo-data:/root/.halo2
    environment:
      - SPRING_DATASOURCE_URL=jdbc:mysql://mysql:3306/halo?characterEncoding=utf8&useSSL=false&serverTimezone=Asia/Shanghai&allowPublicKeyRetrieval=true
      - SPRING_DATASOURCE_USERNAME=halo
      - SPRING_DATASOURCE_PASSWORD=your_strong_password_here  # 修改为强密码
      - SPRING_REDIS_HOST=redis
      - SPRING_REDIS_PORT=6379
      - SPRING_REDIS_PASSWORD=your_redis_password_here  # 修改为强密码
      - SPRING_REDIS_DATABASE=0
      - SERVER_PORT=8090
      - HALO_EXTERNAL_URL=https://your-domain.com  # 修改为您的域名
      - TZ=Asia/Shanghai
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8090/actuator/health/readiness"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s

  mysql:
    image: mysql:8.0
    container_name: halo-mysql
    restart: always
    networks:
      halo-network:
    ports:
      - "127.0.0.1:3307:3306"  # 只监听本地
    volumes:
      - ./mysql-data:/var/lib/mysql
    environment:
      - MYSQL_ROOT_PASSWORD=your_root_password_here  # 修改为强密码
      - MYSQL_DATABASE=halo
      - MYSQL_USER=halo
      - MYSQL_PASSWORD=your_strong_password_here  # 修改为强密码
      - TZ=Asia/Shanghai
    command:
      - --default-authentication-plugin=mysql_native_password
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
      - --max_connections=1000
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-pyour_root_password_here"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: halo-redis
    restart: always
    networks:
      halo-network:
    ports:
      - "127.0.0.1:6379:6379"  # 只监听本地
    volumes:
      - ./redis-data:/data
    command: redis-server --requirepass your_redis_password_here --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "your_redis_password_here", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

networks:
  halo-network:
    driver: bridge
```

**重要修改项：**
- 所有端口改为 `127.0.0.1:port`，只监听本地
- 修改所有默认密码为强密码
- `HALO_EXTERNAL_URL` 改为您的域名（HTTPS）

### 步骤8：启动服务

```bash
cd /opt/halo-blog

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 步骤9：配置防火墙

```bash
# Ubuntu/Debian (UFW)
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable

# CentOS/RHEL (firewalld)
firewall-cmd --permanent --add-service=ssh
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload
```

---

## 域名解析

### 步骤1：获取服务器IP地址

```bash
# 查看服务器公网IP
curl ifconfig.me
# 或
curl ipinfo.io/ip
```

### 步骤2：配置DNS解析

登录您的域名服务商（如阿里云、腾讯云、Cloudflare等），添加以下DNS记录：

#### 方案A：使用主域名（推荐）

| 记录类型 | 主机记录 | 记录值 | TTL |
|---------|---------|--------|-----|
| A | @ | 您的服务器IP | 600 |
| A | www | 您的服务器IP | 600 |

#### 方案B：使用子域名

| 记录类型 | 主机记录 | 记录值 | TTL |
|---------|---------|--------|-----|
| A | blog | 您的服务器IP | 600 |

### 步骤3：验证DNS解析

```bash
# 在本地电脑执行
ping your-domain.com
# 或
nslookup your-domain.com
```

等待DNS生效（通常需要5-30分钟），确认解析到正确的IP地址。

---

## HTTPS证书配置

### 方案一：使用Let's Encrypt免费证书（推荐）

#### 步骤1：安装Certbot

```bash
# Ubuntu/Debian
apt install certbot python3-certbot-nginx -y

# CentOS/RHEL
yum install certbot python3-certbot-nginx -y
```

#### 步骤2：申请证书

```bash
# 方式1：自动配置Nginx（需要先安装Nginx）
certbot --nginx -d your-domain.com -d www.your-domain.com

# 方式2：仅获取证书（手动配置）
certbot certonly --standalone -d your-domain.com -d www.your-domain.com
```

按照提示输入邮箱并同意服务条款。

#### 步骤3：证书位置

证书将保存在以下位置：
- 证书文件：`/etc/letsencrypt/live/your-domain.com/fullchain.pem`
- 私钥文件：`/etc/letsencrypt/live/your-domain.com/privkey.pem`

#### 步骤4：设置自动续期

```bash
# 测试自动续期
certbot renew --dry-run

# 添加自动续期任务
crontab -e

# 添加以下行（每天凌晨2点检查续期）
0 2 * * * certbot renew --quiet && docker-compose -f /opt/halo-blog/docker-compose.yml restart nginx
```

### 方案二：使用付费证书

如果您使用付费证书（如阿里云、腾讯云SSL证书）：

1. 购买SSL证书
2. 下载证书文件（通常包含 .crt 和 .key 文件）
3. 上传到服务器：`/etc/nginx/ssl/`
4. 配置Nginx使用证书

---

## Nginx反向代理配置

### 步骤1：安装Nginx

```bash
# Ubuntu/Debian
apt install nginx -y

# CentOS/RHEL
yum install nginx -y

# 启动Nginx
systemctl start nginx
systemctl enable nginx
```

### 步骤2：创建Nginx配置文件

```bash
nano /etc/nginx/sites-available/halo-blog
```

添加以下配置：

```nginx
# HTTP重定向到HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name your-domain.com www.your-domain.com;

    # Let's Encrypt验证
    location /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    # 重定向到HTTPS
    location / {
        return 301 https://$server_name$request_uri;
    }
}

# HTTPS配置
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name your-domain.com www.your-domain.com;

    # SSL证书配置
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    # SSL优化配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # 安全头
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # 日志配置
    access_log /var/log/nginx/halo-access.log;
    error_log /var/log/nginx/halo-error.log;

    # 客户端上传大小限制
    client_max_body_size 100m;

    # 反向代理到Halo
    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Port $server_port;

        # WebSocket支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时配置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # 缓存配置
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
        proxy_busy_buffers_size 8k;
    }

    # 静态资源缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

### 步骤3：启用配置

```bash
# 创建软链接
ln -s /etc/nginx/sites-available/halo-blog /etc/nginx/sites-enabled/

# 测试配置
nginx -t

# 重载Nginx
nginx -s reload
```

### 步骤4：验证HTTPS

访问 https://your-domain.com，确认：
- HTTPS正常工作
- 证书有效
- 可以正常访问Halo博客

可以使用以下工具测试：
- https://www.ssllabs.com/ssltest/
- https://tools.keycdn.com/http2-test.html

---

## 安全加固

### 1. SSH安全配置

```bash
# 编辑SSH配置
nano /etc/ssh/sshd_config

# 修改以下配置
Port 22222                    # 修改SSH端口
PermitRootLogin no           # 禁止root登录
PasswordAuthentication no     # 禁用密码登录（使用密钥）
PubkeyAuthentication yes      # 启用密钥认证

# 重启SSH服务
systemctl restart sshd
```

### 2. 配置fail2ban防暴力破解

```bash
# 安装fail2ban
apt install fail2ban -y

# 创建配置文件
nano /etc/fail2ban/jail.local
```

添加以下配置：

```ini
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true
port = 22222
filter = sshd
logpath = /var/log/auth.log
maxretry = 3

[nginx-http-auth]
enabled = true
filter = nginx-http-auth
port = http,https
logpath = /var/log/nginx/error.log
```

```bash
# 启动fail2ban
systemctl start fail2ban
systemctl enable fail2ban
```

### 3. 配置自动安全更新

```bash
# Ubuntu/Debian
apt install unattended-upgrades -y
dpkg-reconfigure -plow unattended-upgrades

# CentOS/RHEL
yum install yum-cron -y
systemctl start yum-cron
systemctl enable yum-cron
```

### 4. 配置防火墙规则

```bash
# 只允许特定IP访问SSH（可选）
ufw allow from your_ip_address to any port 22222

# 限制连接频率
ufw limit 22222/tcp
```

---

## 备份策略

### 1. 数据库备份

创建备份脚本：

```bash
nano /opt/halo-blog/backup-mysql.sh
```

```bash
#!/bin/bash
BACKUP_DIR="/opt/halo-blog/backups"
DATE=$(date +%Y%m%d_%H%M%S)
MYSQL_CONTAINER="halo-mysql"
MYSQL_USER="root"
MYSQL_PASSWORD="your_root_password_here"
MYSQL_DATABASE="halo"

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
docker exec $MYSQL_CONTAINER mysqldump -u$MYSQL_USER -p$MYSQL_PASSWORD $MYSQL_DATABASE > $BACKUP_DIR/halo_db_$DATE.sql

# 压缩备份
gzip $BACKUP_DIR/halo_db_$DATE.sql

# 删除7天前的备份
find $BACKUP_DIR -name "halo_db_*.sql.gz" -mtime +7 -delete

echo "Database backup completed: halo_db_$DATE.sql.gz"
```

```bash
# 添加执行权限
chmod +x /opt/halo-blog/backup-mysql.sh

# 添加到定时任务
crontab -e

# 每天凌晨3点备份
0 3 * * * /opt/halo-blog/backup-mysql.sh >> /opt/halo-blog/backup.log 2>&1
```

### 2. 文件备份

```bash
nano /opt/halo-blog/backup-files.sh
```

```bash
#!/bin/bash
BACKUP_DIR="/opt/halo-blog/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份Halo数据
tar -czf $BACKUP_DIR/halo_data_$DATE.tar.gz /opt/halo-blog/halo-data

# 备份Nginx配置
tar -czf $BACKUP_DIR/nginx_config_$DATE.tar.gz /etc/nginx

# 删除7天前的备份
find $BACKUP_DIR -name "halo_data_*.tar.gz" -mtime +7 -delete
find $BACKUP_DIR -name "nginx_config_*.tar.gz" -mtime +7 -delete

echo "Files backup completed"
```

```bash
# 添加执行权限
chmod +x /opt/halo-blog/backup-files.sh

# 添加到定时任务
crontab -e

# 每周日凌晨4点备份
0 4 * * 0 /opt/halo-blog/backup-files.sh >> /opt/halo-blog/backup.log 2>&1
```

### 3. 远程备份（可选）

使用rsync同步到远程服务器：

```bash
# 安装rsync
apt install rsync -y

# 配置远程备份
rsync -avz -e "ssh -p 22222" /opt/halo-blog/backups/ user@remote_server:/backups/halo-blog/
```

---

## 监控和维护

### 1. 监控服务状态

```bash
# 创建监控脚本
nano /opt/halo-blog/monitor.sh
```

```bash
#!/bin/bash
# 检查Docker服务状态
docker-compose -f /opt/halo-blog/docker-compose.yml ps

# 检查Nginx状态
systemctl status nginx

# 检查磁盘空间
df -h

# 检查内存使用
free -h
```

### 2. 日志管理

```bash
# 查看Halo日志
docker-compose -f /opt/halo-blog/docker-compose.yml logs -f halo

# 查看Nginx日志
tail -f /var/log/nginx/halo-access.log
tail -f /var/log/nginx/halo-error.log

# 日志轮转配置
nano /etc/logrotate.d/halo-blog
```

```
/var/log/nginx/halo-*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 www-data adm
    sharedscripts
    postrotate
        [ -f /var/run/nginx.pid ] && kill -USR1 `cat /var/run/nginx.pid`
    endscript
}
```

### 3. 性能优化

```bash
# 优化Nginx配置
nano /etc/nginx/nginx.conf
```

```nginx
user www-data;
worker_processes auto;
worker_rlimit_nofile 65535;

events {
    worker_connections 4096;
    use epoll;
    multi_accept on;
}

http {
    # 基础配置
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # 性能优化
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    client_max_body_size 100m;

    # Gzip压缩
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss application/rss+xml font/truetype font/opentype application/vnd.ms-fontobject image/svg+xml;

    # 包含站点配置
    include /etc/nginx/sites-enabled/*;
}
```

---

## 部署检查清单

完成部署后，请检查以下项目：

- [ ] 服务器已安装Docker和Docker Compose
- [ ] docker-compose.yml已修改为生产环境配置
- [ ] 所有默认密码已修改为强密码
- [ ] 域名DNS解析已生效
- [ ] SSL证书已申请并配置
- [ ] Nginx反向代理已配置
- [ ] HTTP自动重定向到HTTPS
- [ ] 防火墙规则已配置
- [ ] SSH安全配置已完成
- [ ] 备份脚本已配置并测试
- [ ] 监控脚本已配置
- [ ] 服务可以正常访问（https://your-domain.com）
- [ ] Halo后台可以正常访问
- [ ] 数据库备份正常运行
- [ ] SSL证书自动续期已配置

---

## 常见问题

### 1. 域名解析不生效

**解决方案：**
- 检查DNS记录是否正确
- 等待DNS传播（最多48小时）
- 清除本地DNS缓存：`ipconfig /flushdns`（Windows）或 `sudo systemd-resolve --flush-caches`（Linux）

### 2. SSL证书申请失败

**解决方案：**
- 确认域名已正确解析到服务器
- 检查防火墙是否开放80和443端口
- 确认Nginx已启动并监听80端口

### 3. Nginx 502错误

**解决方案：**
- 检查Halo容器是否运行：`docker ps`
- 查看Halo日志：`docker logs halo`
- 检查Nginx配置中的代理地址是否正确

### 4. 无法访问网站

**解决方案：**
- 检查防火墙规则
- 检查Nginx状态：`systemctl status nginx`
- 查看Nginx错误日志：`tail -f /var/log/nginx/halo-error.log`
- 检查SSL证书是否过期

---

## 相关链接

- [Halo官方文档](https://docs.halo.run)
- [Let's Encrypt官网](https://letsencrypt.org/)
- [Nginx官方文档](https://nginx.org/en/docs/)
- [Docker官方文档](https://docs.docker.com/)

---

## 技术支持

如遇到问题，可以：
1. 查看 [Halo社区](https://bbs.halo.run)
2. 访问 [Halo官方文档](https://docs.halo.run)
3. 查看服务器日志排查问题

---

祝您部署顺利！🎉
