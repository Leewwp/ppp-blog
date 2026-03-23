# Halo 稳健性验证 - 面试回答指南

## Q: "你如何验证代码改动的效果和稳健性？"

### 标准回答模板

**当我对代码进行任何修改后，我会按照以下系统化的验证流程确保改动正确且不会引入问题：**

---

### 第一阶段：快速反馈 (5-10分钟)

```bash
# 1. 单元测试 - 验证单个组件
./gradlew :api:test :application:test

# 2. 集成测试 - 验证组件交互
./gradlew :application:test --tests '*IntegrationTest'

# 3. API验证 - 验证接口正确性
./e2e/scripts/validate-apis.sh all
```

**面试追问1："单元测试覆盖率要求是多少？"**

**回答：**
```
项目要求 80%+ 的代码覆盖率。我使用 Jacoco 生成报告：

./gradlew :application:jacocoTestReport

然后查看报告：
open application/build/reports/jacoco/test/html/index.html

关键指标：
- 行覆盖率 > 80%
- 分支覆盖率 > 70%
- 对于核心业务逻辑（如支付、权限），覆盖率要求 > 90%
```

---

### 第二阶段：完整验证 (45-60分钟)

```bash
# 完整流水线
./e2e/scripts/run-all-tests.sh all
```

**面试追问2："如果单元测试和集成测试都通过了，你还需要做哪些验证？"**

**回答：**
```
除了单元和集成测试，我还进行：

1. E2E 测试 - 验证完整用户流程
   ./e2e/start.sh
   覆盖场景：登录 -> 创建文章 -> 浏览 -> 删除

2. 负载测试 - 验证性能
   k6 run e2e/load-tests/api-load.js
   关键指标：
   - P95 延迟 < 500ms
   - 错误率 < 1%
   - 吞吐量 > 1000 RPS

3. 压力测试 - 找到系统极限
   k6 run e2e/load-tests/stress-test.js
   目标：找到系统崩溃点，记录最大并发数
```

---

### 第三阶段：API 合约验证

**面试追问3："如何确保你的 API 改动不会破坏其他服务？"**

**回答：**
```
1. OpenAPI 规范验证
   ./e2e/scripts/validate-apis.sh openapi

   这会验证：
   - 所有端点是否响应正确
   - 响应格式是否符合规范
   - 字段类型是否正确

2. 向后兼容性检查
   - 检查新增字段是否为可选
   - 检查删除字段是否已废弃
   - 使用版本号管理 breaking changes
```

---

### 完整验证链路图

```
代码改动
    │
    ▼
┌─────────────────┐
│  单元测试        │  2-5 分钟
│  (./gradlew :api:test)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  集成测试        │  5-10 分钟
│  (./gradlew :application:test --tests '*IntegrationTest')│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  E2E 测试        │  10-15 分钟
│  (./e2e/start.sh)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  负载测试        │  15-20 分钟
│  (k6 run api-load.js)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  压力测试        │  15-20 分钟
│  (k6 run stress-test.js)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  API 验证        │  2-5 分钟
│  (./validate-apis.sh)│
└────────┬────────┘
         │
         ▼
     验证完成
```

---

## Q: "如何进行压力测试？"

### 回答

```bash
# 1. 首先确保 Halo 运行在目标机器上
./gradlew :application:bootRun &

# 2. 等待启动
sleep 30

# 3. 运行压力测试
k6 run e2e/load-tests/stress-test.js

# 测试会分阶段增加负载：
# Phase 1: 10 用户 (1分钟)
# Phase 2: 50 用户 (2分钟)
# Phase 3: 100 用户 (2分钟)
# Phase 4: 200 用户 (3分钟) ← 预期在这里开始出现性能下降
# Phase 5: 500 用户 (3分钟) ← 预期系统开始报错
# Phase 6: 10 用户恢复检查 (2分钟)

# 4. 查看结果
cat stress-test-results.json | jq '.metrics'
```

**面试追问："如何确定系统的最大承载量？"**

**回答：**
```
通过压力测试逐步增加负载，直到：
1. 错误率超过 5%
2. P95 响应时间超过 2 秒
3. 系统开始拒绝连接

记录此时的用户数（RPS）作为基准线。

建议：
- 正式环境预留 50% 余量
- 监控 CPU、内存、数据库连接数
- 测试不同维度的瓶颈（CPU密集型 vs IO密集型）
```

---

## Q: "如何验证 API 的正确性？"

### 回答

```bash
# 快速健康检查
./e2e/scripts/validate-apis.sh health

# 完整 API 验证
./e2e/scripts/validate-apis.sh all

# 针对特定环境的验证
HALO_URL=http://staging:8090 ./e2e/scripts/validate-apis.sh all
```

**验证内容：**
```
✓ Actuator 健康检查
✓ Public API (Posts, Categories, Tags)
✓ Console API (需要认证)
✓ OpenAPI 规范完整性
```

---

## Q: "如果测试失败了你怎么办？"

### 回答

```bash
# 1. 查看详细日志
./gradlew :application:test --stacktrace

# 2. 单个测试调试
./gradlew :application:test --tests '*SpecificTest*' --debug

# 3. 本地重现问题
./gradlew :application:bootRun
# 然后手动调用 API 测试

# 4. 查看测试报告
open application/build/reports/tests/test/index.html
```

**处理流程：**
```
1. 识别失败的测试是新增代码引起的还是已有问题
2. 如果是新增代码引起：
   - 检查测试假设是否正确
   - 检查实现是否有 bug
3. 如果是已有问题：
   - 检查是否是环境问题
   - 检查是否是测试本身的 flakiness
4. 修复后重新运行测试
```

---

## 快速参考命令

```bash
# 快速验证 (15分钟)
make test-quick

# 完整流水线 (45-60分钟)
make test-all

# 单个测试
make test                    # 单元测试
make test-integration       # 集成测试
make test-load              # 负载测试
make test-api               # API验证

# 直接运行
./e2e/scripts/quick-validate.sh           # 快速验证
./e2e/scripts/run-all-tests.sh all        # 完整流水线
./e2e/scripts/validate-apis.sh health     # 健康检查
```
