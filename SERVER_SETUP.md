# 服务器配置详细指南

本指南将帮助您完成从零开始的服务器配置，包括连接服务器、安装环境、配置项目等。

## 目录

1. [连接服务器](#连接服务器)
2. [安装基础环境](#安装基础环境)
3. [克隆项目](#克隆项目)
4. [配置环境变量](#配置环境变量)
5. [启动服务](#启动服务)
6. [配置Nginx和SSL](#配置nginx和ssl)
7. [配置GitHub Secrets](#配置github-secrets)
8. [测试自动部署](#测试自动部署)

---

## 连接服务器

### 方式1：使用SSH密钥连接（推荐）

您已经有 `ppp_blog.pem` 密钥文件，使用以下命令连接：

```bash
# 在本地终端（Windows PowerShell）执行
ssh -i /path/to/ppp_blog.pem root@your_server_ip
```

**说明**：
- `/path/to/ppp_blog.pem` - 您的密钥文件完整路径
- `your_server_ip` - 您的腾讯云服务器公网IP地址
- `root` - 登录用户名（腾讯云默认为root）

**示例**：
```bash
ssh -i C:\Users\16278\Desktop\file\ppp_blog.pem root@123.456.789.012
```

### 方式2：使用密码连接

如果您设置了服务器密码：

```bash
ssh root@your_server_ip
```

输入密码后即可连接。

---

## 安装基础环境

### 步骤1：更新系统

**在服务器终端**执行：

```bash
# 更新软件包列表
apt update

# 升级已安装的软件包
apt upgrade -y
```

### 步骤2：安装Git

```bash
# 安装Git
apt install git -y

# 验证安装
git --version
```

### 步骤3：安装Docker

```bash
# 使用官方脚本安装Docker
curl -fsSL https://get.docker.com | sh

# 启动Docker服务
systemctl start docker

# 设置Docker开机自启
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

### 步骤5：安装Nginx

```bash
# 安装Nginx
apt install nginx -y

# 启动Nginx
systemctl start nginx

# 设置Nginx开机自启
systemctl enable nginx

# 验证安装
nginx -v
```

### 步骤6：安装其他必要工具

```bash
# 安装Certbot（用于申请SSL证书）
apt install certbot python3-certbot-nginx -y

# 安装文本编辑器（如果需要）
apt install nano -y
```

---

## 克隆项目

### 步骤1：创建项目目录

```bash
# 创建项目目录
mkdir -p /opt
cd /opt
```

### 步骤2：克隆GitHub仓库

```bash
# 克隆您的GitHub仓库
git clone https://github.com/Leewwp/ppp-blog.git halo-blog

# 进入项目目录
cd halo-blog
```

**说明**：
- 使用 `https://` 协议克隆（首次克隆需要）
- 克隆后项目会在 `/opt/halo-blog/` 目录下

### 步骤3：验证项目文件

```bash
# 查看项目文件
ls -la

# 应该看到以下文件：
# - docker-compose.yml
# - deploy.sh
# - .env.example
# - .github/workflows/deploy.yml
# - README.md
# - DEPLOYMENT_GUIDE.md
# - AUTO_DEPLOYMENT.md
```

---

## 配置环境变量

### 步骤1：复制环境变量示例文件

```bash
# 在服务器终端执行
cd /opt/halo-blog
cp .env.example .env
```

### 步骤2：编辑环境变量文件

```bash
# 使用nano编辑器打开
nano .env
```

### 步骤3：修改环境变量

修改以下内容（根据您的实际情况）：

```bash
# MySQL 配置
MYSQL_ROOT_PASSWORD=YourStrongRootPassword123!  # 修改为强密码
MYSQL_DATABASE=halo
MYSQL_USER=halo
MYSQL_PASSWORD=YourStrongMySQLPassword123!  # 修改为强密码

# Redis 配置
REDIS_PASSWORD=YourStrongRedisPassword123!  # 修改为强密码

# Halo 配置
HALO_EXTERNAL_URL=https://your-domain.com  # 修改为您的域名

# 服务器配置
SERVER_HOST=123.456.789.012  # 修改为您的服务器IP
SSH_PORT=22

# 备份配置
BACKUP_RETENTION_DAYS=7
```

**密码安全要求**：
- 长度至少16位
- 包含大小写字母
- 包含数字
- 包含特殊字符（如 !@#$%^&*）

### 步骤4：保存并退出

```bash
# 在nano中按 Ctrl+O 保存
# 按 Ctrl+X 退出
```

---

## 启动服务

### 步骤1：启动Docker服务

```bash
# 在服务器终端执行
cd /opt/halo-blog

# 启动所有服务
docker-compose up -d
```

**首次启动说明**：
- 会自动下载Docker镜像（Halo、MySQL、Redis）
- 需要几分钟时间，请耐心等待
- 可以使用 `docker-compose logs -f` 查看下载进度

### 步骤2：查看服务状态

```bash
# 查看所有容器状态
docker-compose ps
```

**预期输出**：
```
NAME         IMAGE               COMMAND                   SERVICE   CREATED         STATUS                    PORTS
halo         halohub/halo:2.22   "sh -c 'java ${JVM_O…"   halo       2 minutes ago   Up 2 minutes (healthy)   127.0.0.1:8090->8090/tcp
halo-mysql   mysql:8.0           "docker-entrypoint.s…"   mysql      2 minutes ago   Up 2 minutes (healthy)   127.0.0.1:3307->3306/tcp
halo-redis   redis:7-alpine      "docker-entrypoint.s…"   redis      2 minutes ago   Up 2 minutes (healthy)   127.0.0.1:6379->6379/tcp
```

### 步骤3：查看服务日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f halo
docker-compose logs -f mysql
docker-compose logs -f redis
```

### 步骤4：测试Halo服务

```bash
# 测试Halo服务是否正常运行
curl http://localhost:8090/actuator/health/readiness

# 预期输出：{"status":"UP"}
```

---

## 配置Nginx和SSL

### 步骤1：配置域名DNS解析

在您的域名服务商（如阿里云、腾讯云）添加DNS记录：

| 记录类型 | 主机记录 | 记录值 | TTL |
|---------|---------|--------|-----|
| A | @ | 您的服务器IP | 600 |
| A | www | 您的服务器IP | 600 |

**等待DNS生效**：通常需要5-30分钟，可以使用以下命令验证：

```bash
# 在本地电脑执行
ping your-domain.com
```

### 步骤2：申请SSL证书

```bash
# 在服务器终端执行
certbot --nginx -d your-domain.com -d www.your-domain.com
```

按照提示：
1. 输入您的邮箱地址
2. 同意服务条款
3. 选择是否重定向HTTP到HTTPS

### 步骤3：创建Nginx配置文件

```bash
# 创建Nginx配置文件
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

**重要修改项**：
- `server_name`: 改为您的域名
- `ssl_certificate`: 改为您的证书路径
- `ssl_certificate_key`: 改为您的私钥路径

### 步骤4：启用Nginx配置

```bash
# 创建软链接
ln -s /etc/nginx/sites-available/halo-blog /etc/nginx/sites-enabled/

# 测试配置
nginx -t

# 重载Nginx
nginx -s reload
```

### 步骤5：配置防火墙

```bash
# 配置UFW防火墙
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP
ufw allow 443/tcp   # HTTPS
ufw enable
```

---

## 配置GitHub Secrets

### 步骤1：生成SSH密钥对

**在本地电脑**执行：

```bash
# 生成SSH密钥对
ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy

# 查看公钥
cat ~/.ssh/github_actions_deploy.pub
```

### 步骤2：配置服务器SSH公钥

复制公钥内容，然后在**服务器终端**执行：

```bash
# 添加公钥到服务器
echo "your_public_key_here" >> ~/.ssh/authorized_keys

# 设置权限
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### 步骤3：在GitHub中配置Secrets

1. 访问：https://github.com/Leewwp/ppp-blog/settings/secrets/actions
2. 点击 **New repository secret**，添加以下Secrets：

| Secret名称 | 说明 | 示例值 |
|-----------|------|---------|
| `SERVER_HOST` | 服务器IP地址 | `123.456.789.012` |
| `SERVER_USER` | 服务器用户名 | `root` |
| `SSH_PRIVATE_KEY` | SSH私钥内容 | 复制 `~/.ssh/github_actions_deploy` 的完整内容 |
| `SSH_PORT` | SSH端口 | `22` |

**重要**：
- `SSH_PRIVATE_KEY` 的值是完整的私钥文件内容，包括 `-----BEGIN RSA PRIVATE KEY-----` 和 `-----END RSA PRIVATE KEY-----`
- 不要在Secret名称中添加空格
- 每个Secret单独添加

### 步骤4：设置部署脚本权限

```bash
# 在服务器终端执行
cd /opt/halo-blog

# 添加执行权限
chmod +x deploy.sh

# 创建日志目录
mkdir -p /var/log
touch /var/log/halo-deploy.log
chmod 644 /var/log/halo-deploy.log
```

---

## 测试自动部署

### 步骤1：手动触发GitHub Actions

1. 访问：https://github.com/Leewwp/ppp-blog/actions
2. 选择 **Deploy to Server** workflow
3. 点击 **Run workflow** 按钮
4. 选择分支，点击 **Run workflow**

### 步骤2：查看部署日志

在GitHub Actions页面查看执行日志，确认：
- SSH连接成功
- 代码拉取成功
- Docker服务重启成功
- 部署完成

### 步骤3：验证服务状态

```bash
# 在服务器终端执行
docker-compose ps

# 查看部署日志
tail -f /var/log/halo-deploy.log
```

---

## 验证部署

### 步骤1：访问博客

在浏览器中访问：
- **博客前台**: https://your-domain.com
- **博客后台**: https://your-domain.com/console

### 步骤2：检查HTTPS

访问：https://www.ssllabs.com/ssltest/analyze.html?d=your-domain.com

确认：
- SSL证书有效
- HTTPS配置正确
- 安全评级良好

### 步骤3：测试自动部署

在本地修改 `docker-compose.yml`，然后推送：

```bash
git add docker-compose.yml
git commit -m "test: 测试自动部署"
git push origin main
```

等待2-3分钟，确认GitHub Actions自动部署成功。

---

## 常见问题

### 问题1：SSH连接失败

**错误信息**：`Permission denied (publickey)`

**解决方案**：
1. 确认密钥文件路径正确
2. 确认密钥文件权限：`chmod 400 ppp_blog.pem`
3. 确认服务器上已添加对应的公钥

### 问题2：Docker服务启动失败

**错误信息**：`docker-compose up -d` 失败

**解决方案**：
```bash
# 查看详细错误
docker-compose logs

# 检查端口是否被占用
netstat -tulpn | grep :8090
netstat -tulpn | grep :3307
netstat -tulpn | grep :6379
```

### 问题3：SSL证书申请失败

**错误信息**：`Failed to connect to host for DVS`

**解决方案**：
1. 确认域名已正确解析到服务器
2. 检查防火墙是否开放80和443端口
3. 确认Nginx已启动并监听80端口

### 问题4：GitHub Actions部署失败

**错误信息**：`Permission denied (publickey)`

**解决方案**：
1. 检查GitHub Secrets中的 `SSH_PRIVATE_KEY` 是否正确
2. 确认服务器上的 `~/.ssh/authorized_keys` 包含对应的公钥
3. 检查SSH端口是否正确

---

## 配置检查清单

### 服务器配置
- [ ] 已成功连接服务器
- [ ] 已安装Git
- [ ] 已安装Docker
- [ ] 已安装Docker Compose
- [ ] 已安装Nginx
- [ ] 已安装Certbot

### 项目配置
- [ ] 已克隆GitHub仓库到服务器
- [ ] 已配置环境变量（.env文件）
- [ ] 已启动Docker服务
- [ ] 已设置部署脚本权限

### 网络配置
- [ ] 域名DNS解析已生效
- [ ] SSL证书已申请
- [ ] Nginx反向代理已配置
- [ ] 防火墙规则已配置

### 自动化部署
- [ ] 已生成SSH密钥对
- [ ] 已配置服务器SSH公钥
- [ ] 已在GitHub中配置Secrets
- [ ] 已测试GitHub Actions部署
- [ ] 自动部署成功

### 验证测试
- [ ] 博客前台可以正常访问
- [ ] 博客后台可以正常访问
- [ ] HTTPS证书有效
- [ ] 自动部署功能正常

---

## 相关链接

- [腾讯云控制台](https://console.cloud.tencent.com/)
- [Halo官方文档](https://docs.halo.run)
- [Nginx官方文档](https://nginx.org/en/docs/)
- [Let's Encrypt官网](https://letsencrypt.org/)

---

## 技术支持

如遇到问题，可以：
1. 查看 [AUTO_DEPLOYMENT.md](AUTO_DEPLOYMENT.md) - 自动化部署指南
2. 查看 [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) - 手动部署指南
3. 访问 [Halo社区](https://bbs.halo.run)
4. 查看服务器日志排查问题

---

祝您配置顺利！🚀
