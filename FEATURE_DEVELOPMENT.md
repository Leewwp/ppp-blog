# Halo 功能开发指南

本指南将帮助您了解如何在 Halo 博客系统上添加新功能，包括视频音频播放、在线用户统计、数据看板等功能。

## 目录

1. [Halo架构介绍](#halo架构介绍)
2. [功能开发方式对比](#功能开发方式对比)
3. [方式一：开发Halo插件](#方式一开发halo插件推荐)
4. [方式二：开发主题功能](#方式二开发主题功能前端功能推荐)
5. [方式三：修改Halo源码](#方式三修改halo源码高级用户)
6. [具体功能实现方案](#具体功能实现方案)
7. [代码同步机制](#代码同步机制)

---

## Halo架构介绍

### 技术栈

```
Halo 2.x 架构
┌─────────────────────────────────────────────┐
│           前端（Vue 3 + TypeScript）      │
├─────────────────────────────────────────────┤
│           后端（Spring Boot + Java）      │
├─────────────────────────────────────────────┤
│           数据库（MySQL 8.0）            │
├─────────────────────────────────────────────┤
│           缓存（Redis）                   │
└─────────────────────────────────────────────┘
```

### 扩展机制

Halo 提供了多种扩展方式：

1. **插件系统**（Plugin System）
   - 官方推荐的开发方式
   - 可以添加后端功能
   - 支持数据库扩展
   - 提供 API 接口

2. **主题系统**（Theme System）
   - 修改前端展示
   - 添加自定义组件
   - 不影响后端逻辑

3. **扩展点**（Extension Points）
   - 提供钩子（Hooks）
   - 自定义端点（Endpoints）
   - 事件监听（Event Listeners）

---

## 功能开发方式对比

| 开发方式 | 适用场景 | 技术要求 | 代码同步 | 难度 | 推荐度 |
|---------|---------|---------|---------|------|--------|
| **开发Halo插件** | 后端功能、API扩展 | Java + Spring Boot | ✅ Git同步 | 中等 | ⭐⭐⭐⭐⭐ |
| **开发主题功能** | 前端功能、UI定制 | Vue 3 + TypeScript | ✅ Git同步 | 简单 | ⭐⭐⭐⭐ |
| **修改Halo源码** | 深度定制、核心功能修改 | Java + Spring Boot | ✅ Git同步 | 困难 | ⭐⭐ |
| **使用第三方服务** | 数据统计、监控等 | 任意语言 | ❌ 独立部署 | 简单 | ⭐⭐⭐⭐ |

---

## 方式一：开发Halo插件（推荐）

### 适用场景

- 需要添加后端功能
- 需要扩展数据库
- 需要提供 API 接口
- 需要监听 Halo 事件

### 开发流程

#### 步骤1：创建插件项目

```bash
# 使用Halo插件脚手架
git clone https://github.com/halo-dev/plugin-starter.git my-plugin

# 或使用Maven模板
mvn archetype:generate \
  -DarchetypeGroupId=run.halo.app \
  -DarchetypeArtifactId=plugin-archetype \
  -DarchetypeVersion=2.17 \
  -DgroupId=com.example \
  -DartifactId=my-plugin \
  -Dversion=1.0.0-SNAPSHOT
```

#### 步骤2：开发插件功能

插件项目结构：

```
my-plugin/
├── src/
│   ├── main/
│   │   ├── java/
│   │   │   └── com/example/myplugin/
│   │   │       ├── MyPlugin.java          # 插件主类
│   │   │       ├── MyExtension.java      # 扩展点实现
│   │   │       └── MyController.java    # API控制器
│   │   └── resources/
│   │       ├── application.yaml          # 插件配置
│   │       ├── plugin.yaml              # 插件元数据
│   │       └── db/                    # 数据库迁移脚本
├── pom.xml                          # Maven配置
└── README.md
```

**示例：开发在线用户统计插件**

```java
package com.example.myplugin;

import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import run.halo.app.extensionpoint.Extension;
import run.halo.app.extensionpoint.ExtensionPointDefinition;
import run.halo.app.extensionpoint.ExtensionPointDefinitionRegistry;
import run.halo.app.plugin.PluginContext;
import run.halo.app.plugin.ReactiveHaloPlugin;

@Slf4j
@Component
public class MyPlugin extends ReactiveHaloPlugin {

    @Override
    public void onStart(PluginContext context) {
        log.info("MyPlugin started");
        
        // 注册在线用户统计功能
        registerOnlineUserStats(context);
    }

    @Override
    public void onStop(PluginContext context) {
        log.info("MyPlugin stopped");
    }

    private void registerOnlineUserStats(PluginContext context) {
        // 使用Redis存储在线用户
        // 监听用户登录/登出事件
        // 提供统计API
    }
}
```

#### 步骤3：配置插件

`src/main/resources/plugin.yaml`:

```yaml
apiVersion: halo.run/v1alpha1
kind: Plugin
metadata:
  name: my-plugin
  displayName: "我的插件"
  description: "在线用户统计功能"
  version: 1.0.0
  author:
    name: "Your Name"
    website: "https://your-domain.com"
  license:
    name: "GPL-3.0"
  logo: "/logo.png"
spec:
  provides:
    - group: halo.run
      version: v1alpha1
      extensionPoint:
        name: extension-point
  requires:
    - group: halo.run
      version: ">=2.17.0"
```

#### 步骤4：构建插件

```bash
# 在插件项目目录执行
mvn clean package

# 生成的JAR文件在 target/ 目录下
```

#### 步骤5：安装插件

**方式A：通过Halo后台安装**

1. 访问：https://your-domain.com/console
2. 进入：插件 → 安装
3. 上传JAR文件
4. 点击安装

**方式B：手动安装**

```bash
# 将JAR文件复制到Halo插件目录
cp target/my-plugin-1.0.0.jar /opt/halo-blog/halo-data/plugins/

# 重启Halo服务
docker-compose restart halo
```

### 代码同步机制

```bash
# 1. 在本地开发插件
cd my-plugin
# 修改代码...

# 2. 提交到Git
git add .
git commit -m "feat: 添加在线用户统计功能"
git push origin main

# 3. GitHub Actions自动构建
# 在GitHub仓库中配置Actions，自动构建JAR文件

# 4. 服务器自动更新
# GitHub Actions自动上传新JAR到服务器
# Halo自动加载新插件
```

---

## 方式二：开发主题功能（前端功能推荐）

### 适用场景

- 添加视频/音频播放器
- 自定义UI组件
- 添加前端交互功能
- 不需要后端支持的功能

### 开发流程

#### 步骤1：创建主题项目

```bash
# 基于Joe3.0主题开发
git clone https://github.com/qinhua/halo-theme-joe2.0.git my-theme

# 或从零开始
mkdir my-theme
cd my-theme
# 创建主题结构...
```

#### 步骤2：添加视频播放功能

**方式A：使用第三方播放器**

在主题中集成 APlayer 或 DPlayer：

```html
<!-- 在主题模板中添加 -->
<div id="aplayer"></div>

<script src="https://cdn.jsdelivr.net/npm/aplayer@1.10.1/dist/APlayer.min.js"></script>
<script>
  const ap = new APlayer({
    container: document.getElementById('aplayer'),
    audio: [{
      name: '歌曲名称',
      artist: '歌手',
      url: '/path/to/audio.mp3',
      cover: '/path/to/cover.jpg'
    }]
  });
</script>
```

**方式B：使用Halo扩展点**

在文章中插入视频标签：

```markdown
<!-- 在文章编辑器中使用 -->
<joe-dplayer src="https://example.com/video.mp4"></joe-dplayer>

<!-- 或使用B站视频 -->
<joe-bilibili bvid="BV1xx411c7mG"></joe-bilibili>
```

#### 步骤3：添加数据看板功能

**使用第三方服务集成**：

```html
<!-- 集成百度统计 -->
<script>
var _hmt = _hmt || [];
_hmt.push(['_setAccount', 'your-baidu-id']);
(function() {
  var hm = document.createElement("script");
  hm.src = "https://hm.baidu.com/hm.js?si=" + Math.ceil(Math.random()*2147483647) + "-" + new Date().getTime() + "=" + Math.ceil(Math.random()*2147483647);
  var s = document.getElementsByTagName("script")[0];
  s.parentNode.insertBefore(hm, s);
})();
</script>

<!-- 或集成Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script>
```

**开发自定义看板**：

```vue
<!-- 在主题中创建看板组件 -->
<template>
  <div class="dashboard">
    <div class="stat-card">
      <h3>文章统计</h3>
      <div class="stat-item">
        <span>总文章数</span>
        <strong>{{ postCount }}</strong>
      </div>
      <div class="stat-item">
        <span>总浏览量</span>
        <strong>{{ viewCount }}</strong>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      postCount: 0,
      viewCount: 0
    }
  },
  async mounted() {
    // 调用Halo API获取统计数据
    await this.fetchStats();
  },
  methods: {
    async fetchStats() {
      // 使用Halo的Content API
      const response = await fetch('/apis/api.console.halo.run/v1alpha1/posts');
      const data = await response.json();
      this.postCount = data.total;
    }
  }
}
</script>
```

### 代码同步机制

```bash
# 1. 在本地开发主题
cd my-theme
# 修改主题代码...

# 2. 提交到Git
git add .
git commit -m "feat: 添加视频播放器和数据看板"
git push origin main

# 3. 服务器自动更新
# GitHub Actions自动拉取最新主题代码
# Halo自动重新加载主题
```

---

## 方式三：修改Halo源码（高级用户）

### 适用场景

- 需要修改Halo核心功能
- 需要深度定制后端逻辑
- 需要添加新的数据库表

### 开发流程

#### 步骤1：Fork Halo仓库

1. 访问：https://github.com/halo-dev/halo
2. 点击 **Fork** 按钮
3. Fork到您的GitHub账号下

#### 步骤2：克隆Fork的仓库

```bash
# 克隆您Fork的仓库
git clone https://github.com/your-username/halo.git
cd halo
```

#### 步骤3：创建功能分支

```bash
# 创建功能分支
git checkout -b feature/online-user-stats

# 修改代码...
```

#### 步骤4：修改Halo源码

**示例：添加在线用户统计功能**

```java
// 在Halo源码中添加新的API端点
package run.halo.app.api;

import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

@RestController
@RequestMapping("/apis/api.console.halo.run/v1alpha1")
@RequiredArgsConstructor
public class OnlineUserStatsController {

    private final ReactiveExtensionClient extensionClient;

    @GetMapping("/online-users")
    public Mono<Long> getOnlineUserCount() {
        // 从Redis获取在线用户数
        return extensionClient.getExtension("online-user-stats")
            .flatMap(extension -> {
                // 获取在线用户数
                return Mono.just(extension.getOnlineUserCount());
            });
    }
}
```

#### 步骤5：构建Halo

```bash
# 在Halo项目根目录执行
./gradlew clean build -x test

# 生成的JAR文件在 application/build/libs/ 目录下
```

#### 步骤6：部署自定义Halo

```bash
# 停止服务
docker-compose down

# 备份当前JAR
cp application/build/libs/*.jar backup/

# 复制新JAR
cp application/build/libs/halo-2.22.0.jar /opt/halo-blog/

# 重启服务
docker-compose up -d
```

### 代码同步机制

```bash
# 1. 在本地修改Halo源码
cd halo
# 修改代码...

# 2. 提交到您的Fork仓库
git add .
git commit -m "feat: 添加在线用户统计API"
git push origin feature/online-user-stats

# 3. 在服务器上更新
cd /opt/halo-blog/halo
git fetch origin
git reset --hard origin/main

# 4. 重新构建
./gradlew clean build -x test

# 5. 重启服务
docker-compose restart halo
```

---

## 具体功能实现方案

### 功能1：视频音频播放

**推荐方案**：使用主题开发

```html
<!-- 在主题中集成APlayer -->
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/aplayer@1.10.1/dist/APlayer.min.css">
<script src="https://cdn.jsdelivr.net/npm/aplayer@1.10.1/dist/APlayer.min.js"></script>

<div id="aplayer"></div>

<script>
  const ap = new APlayer({
    container: document.getElementById('aplayer'),
    autoplay: false,
    theme: '#FADFA3',
    loop: 'all',
    preload: 'auto',
    volume: 0.7,
    mutex: true,
    list: [{
      name: '示例歌曲',
      artist: '示例歌手',
      url: 'https://example.com/music.mp3',
      cover: 'https://example.com/cover.jpg'
    }, {
      name: '示例视频',
      artist: '示例作者',
      url: 'https://example.com/video.mp4',
      cover: 'https://example.com/video-cover.jpg'
    }]
  });
</script>
```

**位置**：在主题的 `post.ftl` 或 `index.ftl` 中添加

**同步方式**：
```bash
# 1. 修改主题代码
git add .
git commit -m "feat: 添加APlayer播放器"
git push origin main

# 2. 服务器自动更新
# GitHub Actions自动拉取最新主题代码
```

### 功能2：Redis统计当前在线用户

**推荐方案**：开发Halo插件

```java
// 插件主类
package com.example.onlineuserstats;

import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.ReactiveRedisTemplate;
import org.springframework.stereotype.Component;
import run.halo.app.plugin.ReactiveHaloPlugin;

@Slf4j
@Component
public class OnlineUserStatsPlugin extends ReactiveHaloPlugin {

    private final ReactiveRedisTemplate<String, String> redisTemplate;

    @Override
    public void onStart(PluginContext context) {
        log.info("OnlineUserStatsPlugin started");
        
        // 监听用户登录事件
        // 当用户登录时，添加到Redis
        // 当用户登出时，从Redis移除
    }

    @Override
    public void onStop(PluginContext context) {
        log.info("OnlineUserStatsPlugin stopped");
    }
}
```

**位置**：在插件项目中开发

**同步方式**：
```bash
# 1. 开发插件代码
cd my-plugin
# 修改代码...

# 2. 构建插件
mvn clean package

# 3. 提交到Git
git add .
git commit -m "feat: 实现在线用户统计"
git push origin main

# 4. GitHub Actions自动构建
# 配置Actions自动构建JAR文件

# 5. 服务器自动更新
# GitHub Actions自动上传新JAR到服务器
```

### 功能3：新增数据看板

**推荐方案**：使用主题 + Halo API

```vue
<!-- 在主题中创建看板页面 -->
<template>
  <div class="dashboard">
    <div class="stats-grid">
      <div class="stat-card">
        <h3>📝 文章统计</h3>
        <div class="stat-value">{{ stats.postCount }}</div>
        <div class="stat-label">总文章数</div>
      </div>
      <div class="stat-card">
        <h3>👀 浏览统计</h3>
        <div class="stat-value">{{ stats.viewCount }}</div>
        <div class="stat-label">总浏览量</div>
      </div>
      <div class="stat-card">
        <h3>💬 评论统计</h3>
        <div class="stat-value">{{ stats.commentCount }}</div>
        <div class="stat-label">总评论数</div>
      </div>
      <div class="stat-card">
        <h3>👥 在线用户</h3>
        <div class="stat-value">{{ stats.onlineUserCount }}</div>
        <div class="stat-label">当前在线</div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue';

export default {
  setup() {
    const stats = ref({
      postCount: 0,
      viewCount: 0,
      commentCount: 0,
      onlineUserCount: 0
    });

    onMounted(async () => {
      await fetchStats();
    });

    async function fetchStats() {
      // 调用Halo的Content API
      const [postsRes, commentsRes] = await Promise.all([
        fetch('/apis/api.console.halo.run/v1alpha1/posts'),
        fetch('/apis/api.console.halo.run/v1alpha1/comments')
      ]);
      
      const posts = await postsRes.json();
      const comments = await commentsRes.json();
      
      stats.value.postCount = posts.total;
      stats.value.commentCount = comments.total;
      
      // 获取浏览量（需要配置统计插件）
      stats.value.viewCount = await fetchViewCount();
      
      // 获取在线用户数（需要配置在线用户插件）
      stats.value.onlineUserCount = await fetchOnlineUserCount();
    }

    async function fetchViewCount() {
      // 从统计服务获取浏览量
      // 可以使用百度统计、Google Analytics等
      return 0;
    }

    async function fetchOnlineUserCount() {
      // 从在线用户插件API获取
      const response = await fetch('/apis/plugins/my-plugin/v1alpha1/online-users');
      const data = await response.json();
      return data.count;
    }

    return { stats };
  }
};
</script>
```

**位置**：在主题中创建 `dashboard.ftl` 模板

**同步方式**：
```bash
# 1. 修改主题代码
git add .
git commit -m "feat: 添加数据看板页面"
git push origin main

# 2. 服务器自动更新
# GitHub Actions自动拉取最新主题代码
```

---

## 代码同步机制

### 完整的同步流程

```
本地开发环境
    ↓ git push
GitHub仓库（Leewwp/ppp-blog）
    ↓ 触发 GitHub Actions
GitHub Actions
    ↓ SSH 连接
服务器
    ↓ 执行部署脚本
自动部署
    ↓ docker-compose pull
Docker镜像更新
    ↓ docker-compose up -d
服务重启
    ↓
Halo博客
    ├── Halo版本更新（如果修改了docker-compose.yml）
    ├── 主题代码更新（如果修改了主题）
    ├── 插件JAR更新（如果修改了插件）
    └── 配置文件更新（如果修改了配置）
```

### 不同类型的代码同步

| 代码类型 | 存储位置 | 同步方式 | 自动部署 |
|---------|---------|---------|---------|
| **docker-compose.yml** | Git仓库 | ✅ git push | ✅ GitHub Actions |
| **主题代码** | Git仓库 | ✅ git push | ✅ GitHub Actions |
| **插件代码** | Git仓库 | ✅ git push | ✅ GitHub Actions |
| **Halo源码修改** | Fork的Halo仓库 | ✅ git push | ❌ 需要手动构建 |
| **博客内容** | halo-data/db/ | ❌ 不同步 | ❌ Halo后台操作 |

### 实际操作示例

#### 场景1：添加视频播放功能（主题开发）

```bash
# 1. 在本地修改主题代码
cd my-theme
# 添加APlayer代码...

# 2. 提交到Git
git add .
git commit -m "feat: 添加APlayer视频播放器"
git push origin main

# 3. 等待自动部署（2-3分钟）
# GitHub Actions自动拉取最新主题代码
# Halo自动重新加载主题

# 4. 验证功能
# 访问博客，确认视频播放器正常工作
```

#### 场景2：添加在线用户统计（插件开发）

```bash
# 1. 在本地开发插件
cd my-plugin
# 添加在线用户统计代码...

# 2. 构建插件
mvn clean package

# 3. 提交到Git
git add .
git commit -m "feat: 添加在线用户统计功能"
git push origin main

# 4. 等待自动构建
# GitHub Actions自动构建JAR文件

# 5. 等待自动部署
# GitHub Actions自动上传新JAR到服务器
# Halo自动加载新插件

# 6. 验证功能
# 访问博客后台，确认插件已安装
# 测试在线用户统计功能
```

#### 场景3：添加数据看板（主题 + API）

```bash
# 1. 在本地修改主题代码
cd my-theme
# 添加数据看板页面...

# 2. 提交到Git
git add .
git commit -m "feat: 添加数据看板页面"
git push origin main

# 3. 等待自动部署
# GitHub Actions自动拉取最新主题代码
# Halo自动重新加载主题

# 4. 验证功能
# 访问博客，确认数据看板正常显示
```

---

## 推荐的开发路径

### 对于初学者

1. **从主题开发开始**
   - 学习Vue 3和TypeScript
   - 修改现有主题（如Joe3.0）
   - 添加前端功能

2. **使用第三方服务**
   - 使用百度统计、Google Analytics
   - 使用第三方数据看板服务

### 对于进阶开发者

1. **开发Halo插件**
   - 学习Java和Spring Boot
   - 使用Halo插件脚手架
   - 实现后端功能

2. **Fork Halo源码**
   - 深度定制Halo功能
   - 修改核心逻辑

### 对于高级开发者

1. **贡献到Halo社区**
   - 提交Pull Request到官方仓库
   - 参与Halo生态建设

---

## 学习资源

### 官方文档

- [Halo插件开发文档](https://docs.halo.run/developer-guide/plugin/)
- [Halo主题开发文档](https://docs.halo.run/developer-guide/theme/)
- [Halo API文档](https://docs.halo.run/developer-guide/api/)

### 示例项目

- [Halo插件脚手架](https://github.com/halo-dev/plugin-starter)
- [Joe3.0主题](https://github.com/qinhua/halo-theme-joe2.0)
- [Halo官方插件](https://github.com/halo-sigs)

### 社区资源

- [Halo社区](https://bbs.halo.run)
- [Halo Discord](https://discord.gg/halo)
- [Halo Awesome列表](https://github.com/halo-sigs/awesome-halo)

---

## 常见问题

### 问题1：修改主题后没有生效

**解决方案**：
1. 清空浏览器缓存（Ctrl + F5）
2. 在Halo后台重新激活主题
3. 检查主题代码是否有语法错误

### 问题2：插件安装失败

**解决方案**：
1. 检查插件JAR文件是否完整
2. 检查插件版本与Halo版本兼容性
3. 查看Halo日志：`docker logs halo`

### 问题3：代码同步后服务器没有更新

**解决方案**：
1. 检查GitHub Actions是否执行成功
2. 检查服务器上的Git仓库是否是最新的
3. 查看部署日志：`tail -f /var/log/halo-deploy.log`

---

## 总结

### 功能开发方式选择

| 功能类型 | 推荐方式 | 代码位置 | 同步方式 |
|---------|---------|---------|---------|
| **视频/音频播放** | 主题开发 | 主题代码 | ✅ Git同步 |
| **数据看板** | 主题开发 | 主题代码 | ✅ Git同步 |
| **在线用户统计** | 插件开发 | 插件JAR | ✅ Git同步 |
| **后端API扩展** | 插件开发 | 插件JAR | ✅ Git同步 |
| **核心功能修改** | Fork Halo源码 | Halo源码 | ✅ Git同步 |

### 完整的开发流程

```
1. 选择开发方式
   ↓
2. 在本地开发功能
   ↓
3. 提交到Git
   ↓
4. GitHub Actions自动部署
   ↓
5. 服务器自动更新
   ↓
6. 验证功能
```

---

祝您开发顺利！🚀
