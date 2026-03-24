# Halo 博客一站式自动化部署指南

本指南将帮助您配置从本地开发到服务器的完整自动化部署流程，实现：**本地开发 → 推送GitHub → 服务器自动更新**。

## 目录

1. [部署架构](#部署架构)
2. [初始部署](#初始部署)
3. [GitHub Actions配置](#github-actions配置)
4. [自动化部署流程](#自动化部署流程)
5. [日常开发流程](#日常开发流程)
6. [故障排查](#故障排查)
7. [高级配置](#高级配置)

---

## 部署架构

### 工作流程

```
本地开发环境
    ↓ (git push)
GitHub 仓库
    ↓ (触发 GitHub Actions)
GitHub Actions
    ↓ (SSH 连接)
腾讯云服务器
    ↓ (执行部署脚本)
自动部署
```

### 技术栈

- **版本控制**: Git + GitHub
- **CI/CD**: GitHub Actions
- **容器化**: Docker + Docker Compose
- **反向代理**: Nginx
- **SSL证书**: Let's Encrypt

### 文件结构

```
ppp-blog/
├── .github/
│   └── workflows/
│       └── deploy.yml          # GitHub Actions 配置
├── docker-compose.yml           # Docker Compose 配置
├── deploy.sh                  # 服务器端部署脚本
├── .env.example              # 环境变量示例
├── .gitignore               # Git 忽略文件
├── README.md                 # 项目说明
├── DEPLOYMENT_GUIDE.md      # 手动部署指南
└── AUTO_DEPLOYMENT.md       # 本文档（自动化部署）
```

---

## 初始部署

### 步骤1：连接腾讯云服务器

```bash
# 使用SSH连接
ssh root@your_server_ip

# 或使用密钥连接
ssh -i /path/to/your/key.pem root@your_server_ip
```

### 步骤2：安装基础环境

```bash
# 更新系统
apt update && apt upgrade -y

# 安装Git
apt install git -y

# 安装Docker
curl -fsSL https://get.docker.com | sh
systemctl start docker
systemctl enable docker

# 安装Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# 验证安装
docker --version
docker-compose --version
git --version
```

### 步骤3：创建项目目录

```bash
# 创建项目目录
mkdir -p /opt/halo-blog
cd /opt/halo-blog

# 创建数据目录
mkdir -p backups logs
```

### 步骤4：克隆GitHub仓库

```bash
# 克隆您的GitHub仓库
git clone https://github.com/your-username/ppp-blog.git .

# 或使用SSH克隆（推荐）
git clone git@github.com:your-username/ppp-blog.git .
```

### 步骤5：配置环境变量

```bash
# 复制环境变量示例文件
cp .env.example .env

# 编辑环境变量
nano .env
```

修改以下内容：

```bash
# MySQL 配置
MYSQL_ROOT_PASSWORD=YourStrongRootPassword123!
MYSQL_DATABASE=halo
MYSQL_USER=halo
MYSQL_PASSWORD=YourStrongMySQLPassword123!

# Redis 配置
REDIS_PASSWORD=YourStrongRedisPassword123!

# Halo 配置
HALO_EXTERNAL_URL=https://your-domain.com
```

**重要**：所有密码必须是强密码（16位以上，包含大小写字母、数字、特殊字符）

### 步骤6：修改docker-compose.yml

编辑 `docker-compose.yml`，确保配置正确：

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
      - "127.0.0.1:8090:8090"  # 只监听本地
    volumes:
      - ./halo-data:/root/.halo2
    environment:
      - SPRING_DATASOURCE_URL=jdbc:mysql://mysql:3306/halo?characterEncoding=utf8&useSSL=false&serverTimezone=Asia/Shanghai&allowPublicKeyRetrieval=true
      - SPRING_DATASOURCE_USERNAME=halo
      - SPRING_DATASOURCE_PASSWORD=${MYSQL_PASSWORD}
      - SPRING_REDIS_HOST=redis
      - SPRING_REDIS_PORT=6379
      - SPRING_REDIS_PASSWORD=${REDIS_PASSWORD}
      - SPRING_REDIS_DATABASE=0
      - SERVER_PORT=8090
      - HALO_EXTERNAL_URL=${HALO_EXTERNAL_URL}
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
      - MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD}
      - MYSQL_DATABASE=halo
      - MYSQL_USER=halo
      - MYSQL_PASSWORD=${MYSQL_PASSWORD}
      - TZ=Asia/Shanghai
    command:
      - --default-authentication-plugin=mysql_native_password
      - --character-set-server=utf8mb4
      - --collation-server=utf8mb4_unicode_ci
      - --max_connections=1000
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD}"]
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
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

networks:
  halo-network:
    driver: bridge
```

### 步骤7：启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

### 步骤8：配置Nginx和SSL

按照 [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) 中的步骤配置：
- 域名解析
- SSL证书申请
- Nginx反向代理配置

### 步骤9：设置部署脚本权限

```bash
# 添加执行权限
chmod +x /opt/halo-blog/deploy.sh

# 创建日志目录
mkdir -p /var/log
touch /var/log/halo-deploy.log
chmod 644 /var/log/halo-deploy.log
```

---

## GitHub Actions配置

### 步骤1：生成SSH密钥对

在**本地电脑**执行：

```bash
# 生成SSH密钥对
ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy

# 查看公钥
cat ~/.ssh/github_actions_deploy.pub
```

### 步骤2：配置服务器SSH公钥

复制公钥内容，然后在**服务器**上执行：

```bash
# 添加公钥到服务器
echo "your_public_key_here" >> ~/.ssh/authorized_keys

# 设置权限
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
```

### 步骤3：配置GitHub Secrets

1. 访问您的GitHub仓库
2. 点击 **Settings** → **Secrets and variables** → **Actions**
3. 点击 **New repository secret**，添加以下Secrets：

| Secret名称 | 说明 | 示例值 |
|-----------|------|---------|
| `SERVER_HOST` | 服务器IP地址 | `123.456.789.012` |
| `SERVER_USER` | 服务器用户名 | `root` |
| `SSH_PRIVATE_KEY` | SSH私钥内容 | 复制 `~/.ssh/github_actions_deploy` 的内容 |
| `SSH_PORT` | SSH端口 | `22` |

**重要**：
- `SSH_PRIVATE_KEY` 的值是完整的私钥文件内容，包括 `-----BEGIN RSA PRIVATE KEY-----` 和 `-----END RSA PRIVATE KEY-----`
- 不要在Secret名称中添加空格

### 步骤4：验证GitHub Actions配置

1. 在GitHub仓库中，点击 **Actions** 标签
2. 选择 **Deploy to Server** workflow
3. 点击 **Run workflow** 手动触发一次
4. 查看执行日志，确认部署成功

---

## 自动化部署流程

### 触发方式

GitHub Actions支持两种触发方式：

#### 方式1：自动触发（推荐）

当您推送代码到 `main` 或 `master` 分支时，自动触发部署：

```bash
# 在本地开发
git add .
git commit -m "feat: 添加新功能"
git push origin main
```

推送后，GitHub Actions会自动执行部署。

#### 方式2：手动触发

在GitHub仓库中：
1. 点击 **Actions** 标签
2. 选择 **Deploy to Server** workflow
3. 点击 **Run workflow** 按钮
4. 选择分支，点击 **Run workflow**

### 部署流程

GitHub Actions执行以下步骤：

1. **Checkout代码**：从GitHub拉取最新代码
2. **SSH连接服务器**：使用配置的SSH密钥连接服务器
3. **拉取最新代码**：在服务器上执行 `git pull`
4. **重启Docker服务**：执行 `docker-compose down && docker-compose up -d`
5. **清理镜像**：删除未使用的Docker镜像
6. **完成部署**：输出部署成功信息

### 部署脚本说明

服务器端的 `deploy.sh` 脚本执行以下操作：

```bash
1. 备份当前配置
2. 拉取最新代码（git pull）
3. 停止当前服务（docker-compose down）
4. 拉取最新镜像（docker-compose pull）
5. 启动服务（docker-compose up -d）
6. 等待服务启动
7. 检查服务状态
8. 清理未使用的镜像
9. 输出部署日志
```

---

## 日常开发流程

### 典型工作流程

```bash
# 1. 在本地开发新功能
# 修改代码...

# 2. 提交代码
git add .
git commit -m "feat: 添加新功能"

# 3. 推送到GitHub
git push origin main

# 4. 等待自动部署（约2-3分钟）
# 查看GitHub Actions状态
```

### 开发最佳实践

#### 1. 分支管理

```bash
# 创建功能分支
git checkout -b feature/new-feature

# 开发完成后合并到主分支
git checkout main
git merge feature/new-feature
git push origin main
```

#### 2. 本地测试

```bash
# 在本地测试
docker-compose up -d

# 测试完成后停止
docker-compose down
```

#### 3. 代码审查

```bash
# 使用Pull Request进行代码审查
git checkout -b feature/new-feature
# 开发...
git push origin feature/new-feature
# 在GitHub上创建Pull Request
```

---

## 故障排查

### 问题1：GitHub Actions执行失败

**错误信息**：`Permission denied (publickey)`

**解决方案**：
1. 检查SSH私钥是否正确配置在GitHub Secrets中
2. 确认服务器上的 `~/.ssh/authorized_keys` 包含对应的公钥
3. 检查SSH端口是否正确

### 问题2：部署脚本执行失败

**错误信息**：`Please run this script as root`

**解决方案**：
- 确保GitHub Actions配置的 `SERVER_USER` 为 `root`
- 或修改 `deploy.sh` 脚本，移除root检查

### 问题3：Docker服务启动失败

**错误信息**：`docker-compose up -d` 失败

**解决方案**：
```bash
# 在服务器上查看日志
cd /opt/halo-blog
docker-compose logs

# 查看详细错误
docker-compose ps
```

### 问题4：部署后网站无法访问

**解决方案**：
```bash
# 检查服务状态
docker-compose ps

# 检查Nginx状态
systemctl status nginx

# 查看Nginx日志
tail -f /var/log/nginx/halo-error.log
```

### 问题5：数据库连接失败

**解决方案**：
1. 检查 `.env` 文件中的数据库密码是否正确
2. 确认MySQL容器是否正常运行
3. 查看MySQL日志：`docker logs halo-mysql`

---

## 高级配置

### 1. 配置部署通知

#### 钉钉通知

修改 `deploy.sh`，在部署完成后发送通知：

```bash
# 在脚本末尾添加
DINGTALK_WEBHOOK="https://oapi.dingtalk.com/robot/send?access_token=your_token"
curl -X POST "$DINGTALK_WEBHOOK" \
  -H 'Content-Type: application/json' \
  -d "{
    \"msgtype\": \"text\",
    \"text\": {
      \"content\": \"Halo博客部署成功！\n时间: $(date)\n分支: $(git rev-parse --abbrev-ref HEAD)\n提交: $(git rev-parse --short HEAD)\"
    }
  }"
```

#### 企业微信通知

```bash
WECHAT_WEBHOOK="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your_key"
curl -X POST "$WECHAT_WEBHOOK" \
  -H 'Content-Type: application/json' \
  -d "{
    \"msgtype\": \"text\",
    \"text\": {
      \"content\": \"Halo博客部署成功！\"
    }
  }"
```

### 2. 配置零停机部署

修改 `deploy.sh`，实现零停机部署：

```bash
# 拉取最新镜像
docker-compose pull

# 启动新容器（不停止旧容器）
docker-compose up -d --no-deps --build

# 等待新容器就绪
sleep 30

# 停止旧容器
docker-compose down

# 清理旧容器
docker container prune -f
```

### 3. 配置回滚机制

添加回滚脚本：

```bash
nano /opt/halo-blog/rollback.sh
```

```bash
#!/bin/bash
set -e

BACKUP_DIR="/opt/halo-blog/backups"
LATEST_BACKUP=$(ls -t $BACKUP_DIR | head -1)

echo "回滚到: $LATEST_BACKUP"

# 停止当前服务
cd /opt/halo-blog
docker-compose down

# 恢复备份
cp $BACKUP_DIR/$LATEST_BACKUP/docker-compose.yml.*.backup docker-compose.yml
cp $BACKUP_DIR/$LATEST_BACKUP/.env.*.backup .env 2>/dev/null || true

# 重启服务
docker-compose up -d

echo "回滚完成！"
```

```bash
chmod +x /opt/halo-blog/rollback.sh
```

### 4. 配置多环境部署

创建不同的配置文件：

```bash
# 开发环境
docker-compose -f docker-compose.dev.yml up -d

# 生产环境
docker-compose -f docker-compose.prod.yml up -d
```

修改GitHub Actions，支持多环境部署：

```yaml
jobs:
  deploy-dev:
    runs-on: ubuntu-latest
    steps:
      - uses: appleboy/ssh-action@v1.0.0
        with:
          host: ${{ secrets.DEV_SERVER_HOST }}
          script: |
            cd /opt/halo-blog-dev
            ./deploy.sh

  deploy-prod:
    runs-on: ubuntu-latest
    steps:
      - uses: appleboy/ssh-action@v1.0.0
        with:
          host: ${{ secrets.PROD_SERVER_HOST }}
          script: |
            cd /opt/halo-blog
            ./deploy.sh
```

### 5. 配置健康检查

在部署后自动检查服务健康状态：

```bash
# 在deploy.sh末尾添加
echo "检查服务健康状态..."

# 检查Halo服务
for i in {1..30}; do
    if curl -f http://localhost:8090/actuator/health/readiness; then
        echo "Halo服务正常"
        break
    fi
    echo "等待Halo服务启动... ($i/30)"
    sleep 2
done

# 检查Nginx服务
if systemctl is-active --quiet nginx; then
    echo "Nginx服务正常"
else
    echo "Nginx服务异常！"
    exit 1
fi
```

---

## 监控和维护

### 1. 查看部署日志

```bash
# 在服务器上查看部署日志
tail -f /var/log/halo-deploy.log

# 查看最近的部署记录
grep "Deployment completed" /var/log/halo-deploy.log
```

### 2. 监控GitHub Actions

1. 访问GitHub仓库的 **Actions** 页面
2. 查看每次部署的执行状态
3. 点击具体的workflow查看详细日志

### 3. 配置告警

在GitHub Actions中配置失败告警：

```yaml
# 在deploy.yml中添加
- name: Notify on failure
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    text: '部署失败！'
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

---

## 安全建议

### 1. SSH密钥管理

- 使用专用的SSH密钥对用于GitHub Actions
- 定期轮换SSH密钥
- 限制SSH密钥的权限（只读）

### 2. GitHub Secrets管理

- 不要在代码中硬编码敏感信息
- 定期更新Secrets
- 使用强密码

### 3. 服务器安全

- 定期更新系统和软件
- 配置防火墙规则
- 启用fail2ban防暴力破解

---

## 性能优化

### 1. 加速部署

```yaml
# 在deploy.yml中使用缓存
- name: Cache Docker layers
  uses: actions/cache@v3
  with:
    path: /tmp/.buildx-cache
    key: ${{ runner.os }}-buildx-${{ github.sha }}
```

### 2. 并行部署

```yaml
# 同时部署多个服务
jobs:
  deploy:
    strategy:
      matrix:
        service: [halo, mysql, redis]
    runs-on: ubuntu-latest
    steps:
      - name: Deploy ${{ matrix.service }}
        run: |
          echo "Deploying ${{ matrix.service }}"
```

---

## 常见问题FAQ

### Q1: 如何查看部署历史？

**A**: 在GitHub仓库的 **Actions** 页面查看所有部署记录，或在服务器上查看 `/var/log/halo-deploy.log`

### Q2: 如何回滚到上一个版本？

**A**: 
1. 在GitHub上查看commit历史
2. 回滚到指定的commit
3. 或使用服务器上的 `rollback.sh` 脚本

### Q3: 部署需要多长时间？

**A**: 通常需要2-5分钟，取决于：
- 代码大小
- 镜像拉取速度
- 网络状况

### Q4: 如何在部署前测试？

**A**: 
1. 使用Pull Request进行代码审查
2. 在本地测试
3. 使用测试环境

### Q5: 如何配置定时部署？

**A**: 在GitHub Actions中添加定时触发：

```yaml
on:
  schedule:
    - cron: '0 2 * * *'  # 每天凌晨2点
```

---

## 相关链接

- [GitHub Actions文档](https://docs.github.com/en/actions)
- [Docker Compose文档](https://docs.docker.com/compose/)
- [Halo官方文档](https://docs.halo.run)
- [腾讯云文档](https://cloud.tencent.com/document/product/213)

---

## 技术支持

如遇到问题，可以：
1. 查看 [GitHub Actions日志](https://github.com/your-username/ppp-blog/actions)
2. 查看 [服务器部署日志](/var/log/halo-deploy.log)

## Validation-first Pipeline

The deployment flow now assumes two isolated server paths on the Guangzhou host:

- Production: `/opt/halo-blog`
- Validation: `/opt/halo-validation`

The validation instance runs Halo on port `18090`, executes API validation plus k6 load/stress tests, and writes a GitHub Actions Summary before production deployment is allowed to continue.

### Required GitHub Secrets

- `SERVER_HOST`
- `SERVER_USER`
- `SSH_PRIVATE_KEY`
- `SSH_PORT`
- `VALIDATION_MYSQL_ROOT_PASSWORD`
- `VALIDATION_MYSQL_PASSWORD`
- `VALIDATION_REDIS_PASSWORD`

### Required Server Actions

1. Create `/opt/halo-validation`.
2. Run `bash validation/server/setup-validation.sh` inside that directory.
3. Open firewall/security-group port `18090`.
4. Verify the validation URL is reachable from GitHub Actions: `http://<server-ip>:18090/actuator/health/readiness`.

### Production Gating

`.github/workflows/deploy.yml` no longer deploys directly on every push. It now waits for the `Halo Validation Pipeline` workflow to complete successfully, then deploys the validated commit SHA to `/opt/halo-blog`.
3. 访问 [Halo社区](https://bbs.halo.run)

---

## 部署检查清单

完成配置后，请检查以下项目：

### 初始部署
- [ ] 服务器已安装Docker和Docker Compose
- [ ] 已克隆GitHub仓库到服务器
- [ ] 已配置环境变量（.env文件）
- [ ] 已修改docker-compose.yml为生产环境配置
- [ ] 已启动Docker服务
- [ ] 已配置Nginx和SSL证书
- [ ] 已设置部署脚本权限

### GitHub Actions配置
- [ ] 已生成SSH密钥对
- [ ] 已配置服务器SSH公钥
- [ ] 已在GitHub中配置Secrets
- [ ] 已测试GitHub Actions部署
- [ ] 部署日志正常输出

### 自动化部署
- [ ] 推送代码后自动触发部署
- [ ] 部署完成后服务正常运行
- [ ] 部署日志记录正常
- [ ] 已配置部署通知（可选）

---

祝您使用愉快！🚀
