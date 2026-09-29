# 开发与验证

## 1. 编译工具链

```bash
command -v go
command -v clang++
command -v java
command -v scalac
command -v sbt
```

当前 Makefile 直接调用 `go`、`clang++`、`javac`、`scalac`、`scala`。如果工具不在 PATH，应先修复环境，不要把绝对路径硬编码进源码。

## 2. Make 目标

| 命令 | 作用 |
|---|---|
| `make build` | 编译 C++、Go、Java、Scala |
| `make build-cpp` | 编译 `bin/execution_core` |
| `make build-go` | 编译 `bin/control-plane` |
| `make build-java` | 编译 Java class 文件 |
| `make build-scala` | 编译 Scala class 文件 |
| `make test` | 执行 Go 单元测试 |
| `make research` | 运行 Scala 因子和 Java 回测 |
| `make run` | 启动 Go 控制面和 C++ 子进程 |
| `make smoke` | 运行全链路冒烟测试 |
| `make clean` | 删除编译产物和 `data/signals.csv` |

## 3. 验证矩阵

### Go

```bash
make test
cd /Users/jiangdadong/WorkBuddy/2026-09-29-00-21-00/quant-hybrid
go vet ./...
```

当前测试覆盖：

- 合法订单通过；
- 单笔数量超过 100 被拒绝；
- 绝对仓位超过 1000 被拒绝。

### C++

```bash
make build-cpp
printf '%s\n' \
  '{"type":"order","order_id":"cpp-1","symbol":"DEMO","side":"BUY","quantity":3,"limit_price":12.5}' \
  '{"type":"query"}' \
  | ./bin/execution_core
```

应该看到 `accepted`、`fill` 和 `snapshot` 三类事件。

### Scala + Java

```bash
make research
```

应该先生成 `data/signals.csv`，再输出 Java JSON 结果。

### 端到端

```bash
make smoke
```

脚本使用临时端口 `18080`，退出时会清理后台控制面进程。

## 4. 代码修改后的最小流程

```bash
make clean
make build
make test
make research
make smoke
cd /Users/jiangdadong/WorkBuddy/2026-09-29-00-21-00/quant-hybrid
go vet ./...
git diff --check
```

## 5. 调试顺序

### HTTP 返回 503

1. 检查 `bin/execution_core` 是否存在；
2. 直接运行 C++ 核心验证是否能启动；
3. 检查 Go `--engine` 参数；
4. 检查 C++ stdout 是否仍是合法 NDJSON。

### 快照没有持仓

Go 消费 C++ 事件是异步的。先等待 `fill` 事件，再查询 `/snapshot`。生产系统不能靠 sleep，应改成事件确认、sequence 或查询状态机。

### 回测结果异常

1. 检查 CSV 是否按日期升序；
2. 检查价格是否为正数；
3. 检查 signals 是否和 prices 使用相同日期；
4. 确认是否错误地把最后一天当成可计算 T+1 收益；
5. 检查是否把理想成交结果误认为可实现收益。

## 6. 生成文件和忽略规则

以下内容是构建或运行生成物，不应提交：

```text
bin/*
build/java/*
build/scala/*
data/signals.csv
```
