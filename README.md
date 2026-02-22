# Halo 2.x Docker Compose 部署方案

本项目提供了 Halo 2.x 博客系统的完整 Docker Compose 部署方案，包含 Halo 主服务、MySQL 8.0 数据库和 Redis 缓存服务。

## 系统要求

- Docker 24+
- Docker Compose 2.x+
- 至少 2GB 可用内存
- 至少 10GB 可用磁盘空间

## 快速启动

### 1. 启动所有服务

在项目根目录下执行：

```bash
docker-compose up -d
```

首次启动会自动下载所需的 Docker 镜像，可能需要几分钟时间。

### 2. 查看服务状态

```bash
docker-compose ps
```

### 3. 查看服务日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f halo
docker-compose logs -f mysql
docker-compose logs -f redis
```

## 访问服务

启动完成后，可以通过以下地址访问：

- **Halo 前台**: http://localhost:8090
- **Halo 后台**: http://localhost:8090/console
- **MySQL**: localhost:3306
- **Redis**: localhost:6379

首次访问 Halo 后台时，需要进行初始化配置。

## 服务配置

### Halo 服务

- **镜像**: halohub/halo:2.22
- **端口**: 8090
- **数据目录**: ./halo-data
- **工作目录**: /root/.halo2

### MySQL 服务

- **镜像**: mysql:8.0
- **端口**: 3306
- **数据目录**: ./mysql-data
- **数据库名**: halo
- **用户名**: halo
- **密码**: halo_password
- **Root 密码**: root_password

### Redis 服务

- **镜像**: redis:7-alpine
- **端口**: 6379
- **数据目录**: ./redis-data
- **密码**: redis_password

## 数据持久化

所有服务的数据都通过 Docker 卷进行持久化存储：

- `./halo-data`: Halo 应用数据
- `./mysql-data`: MySQL 数据库数据
- `./redis-data`: Redis 缓存数据

**重要**: 请定期备份这些目录，以防数据丢失。

## 常用命令

### 停止服务

```bash
docker-compose stop
```

### 启动服务

```bash
docker-compose start
```

### 重启服务

```bash
docker-compose restart
```

### 停止并删除容器

```bash
docker-compose down
```

### 停止并删除容器及数据卷

```bash
docker-compose down -v
```

### 更新 Halo 版本

```bash
# 拉取最新镜像
docker-compose pull halo

# 重启服务
docker-compose up -d halo
```

## 环境变量说明

### Halo 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| SPRING_DATASOURCE_URL | - | MySQL 数据库连接地址 |
| SPRING_DATASOURCE_USERNAME | halo | 数据库用户名 |
| SPRING_DATASOURCE_PASSWORD | halo_password | 数据库密码 |
| SPRING_REDIS_HOST | redis | Redis 主机地址 |
| SPRING_REDIS_PORT | 6379 | Redis 端口 |
| SPRING_REDIS_PASSWORD | redis_password | Redis 密码 |
| SERVER_PORT | 8090 | Halo 服务端口 |
| HALO_EXTERNAL_URL | http://localhost:8090 | Halo 外部访问地址 |
| TZ | Asia/Shanghai | 时区设置 |

### MySQL 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| MYSQL_ROOT_PASSWORD | root_password | MySQL Root 密码 |
| MYSQL_DATABASE | halo | 数据库名称 |
| MYSQL_USER | halo | 数据库用户名 |
| MYSQL_PASSWORD | halo_password | 数据库密码 |

### Redis 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| redis_password | redis_password | Redis 密码 |

## 安全建议

生产环境部署前，请务必修改以下默认密码：

1. MySQL Root 密码 (MYSQL_ROOT_PASSWORD)
2. MySQL 用户密码 (MYSQL_PASSWORD)
3. Redis 密码 (redis_password)

修改方法：编辑 `docker-compose.yml` 文件中的环境变量，然后重启服务：

```bash
docker-compose down
docker-compose up -d
```

## 故障排查

### 服务无法启动

1. 检查端口是否被占用
2. 查看服务日志: `docker-compose logs [service_name]`
3. 检查磁盘空间是否充足

### 数据库连接失败

1. 确认 MySQL 服务已启动: `docker-compose ps mysql`
2. 检查数据库连接配置
3. 查看 MySQL 日志: `docker-compose logs mysql`

### Redis 连接失败

1. 确认 Redis 服务已启动: `docker-compose ps redis`
2. 检查 Redis 密码配置
3. 查看 Redis 日志: `docker-compose logs redis`

## 备份与恢复

### 备份数据

```bash
# 备份 Halo 数据
tar -czf halo-data-backup-$(date +%Y%m%d).tar.gz ./halo-data

# 备份 MySQL 数据
docker exec halo-mysql mysqldump -uroot -proot_password halo > halo-db-backup-$(date +%Y%m%d).sql

# 备份 Redis 数据
docker exec halo-redis redis-cli -a redis_password BGSAVE
tar -czf redis-data-backup-$(date +%Y%m%d).tar.gz ./redis-data
```

### 恢复数据

```bash
# 恢复 Halo 数据
tar -xzf halo-data-backup-YYYYMMDD.tar.gz

# 恢复 MySQL 数据
docker exec -i halo-mysql mysql -uroot -proot_password halo < halo-db-backup-YYYYMMDD.sql

# 恢复 Redis 数据
tar -xzf redis-data-backup-YYYYMMDD.tar.gz
docker-compose restart redis
```

## 相关链接

- [Halo 官网](https://www.halo.run)
- [Halo 文档](https://docs.halo.run)
- [Halo 社区](https://bbs.halo.run)
- [Halo GitHub](https://github.com/halo-dev/halo)

## 许可证

Halo 使用 GPL-v3.0 协议开源。
