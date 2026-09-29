# quant-hybrid

混合语言量化交易系统 MVP：把实时交易路径和研究回测路径拆开，用各语言做擅长的事情。

> 当前版本只运行本地 CSV 和模拟执行核心，不连接真实交易所、券商或账户，不产生真实订单。

## 1. 系统定位

实时交易路径必须短、稳定、可控：

```text
HTTP /orders
    -> Go control-plane
    -> C++ execution_core
    -> accepted / fill / reject
```

研究路径允许批处理和实验：

```text
prices.csv
    -> Scala FactorJob
    -> signals.csv
    -> Java Backtest
    -> return / drawdown / trades
```

语言职责不是为了凑数，而是明确边界：

| 子系统 | 语言 | 当前职责 | 不负责什么 |
|---|---|---|---|
| 控制面 / OMS | Go | HTTP、订单校验、风险门禁、事件快照、子进程管理 | 不做低延迟撮合 |
| 执行核心 | C++20 | NDJSON 订单解析、模拟成交、现金和持仓更新 | 不提供公网 HTTP |
| 因子批处理 | Scala 2.13 | SMA5/SMA10 因子计算、信号文件生成 | 不进入实时交易路径 |
| 回测引擎 | Java 27 | 下一交易日收益、收益率、最大回撤、交易次数 | 不连接实盘账户 |

## 2. 目录

```text
.
├── README.md                         # 项目总览和快速入口
├── QUICKSTART.txt                    # 最短命令清单
├── Makefile                          # 编译、测试、研究、运行和清理
├── config/system.json                # 当前风险限制和语言边界
├── data/prices.csv                   # 示例行情输入
├── proto/events.proto                # 长期演进的事件协议草案
├── cpp/execution_core/main.cpp       # C++ 模拟执行核心
├── go/control-plane/main.go          # Go 控制面
├── go/control-plane/main_test.go     # Go 风控单测
├── java/backtest/src/Backtest.java   # Java 回测
├── scala/factor/src/main/scala/      # Scala 因子任务
├── scripts/smoke.sh                  # 端到端冒烟测试
└── docs/
    ├── README.md                     # 文档索引
    ├── architecture.md               # 架构和边界
    ├── architecture.html             # 可视化架构页
    ├── api.md                         # HTTP / NDJSON / Protobuf 契约
    ├── research.md                    # 因子和回测口径
    ├── development.md                 # 本地开发和验证
    ├── roadmap.md                     # 生产化演进路线
    └── risk-boundaries.md             # 风险边界和上线前检查
```

## 3. 环境要求

本项目按 macOS/Linux 的命令行工具链设计：

- Go：可编译当前 `go.mod`，推荐 Go 1.26+
- C++：支持 C++20 的 `clang++` 或 `g++`
- Java：JDK 27 EA 用于当前开发验证；代码本身未依赖 JDK 27 专有 API
- Scala：Scala 2.13，当前使用 `scalac` 和 `scala`
- GNU Make：当前 Makefile 使用标准 Make 语法
- `curl`：仅用于 smoke 脚本和本地 HTTP 验证

## 4. 快速开始

```bash
cd /Users/jiangdadong/WorkBuddy/2026-09-29-00-21-00/quant-hybrid

# 编译 Go/C++/Java/Scala
make build

# 运行 Scala 因子 -> Java 回测
make research

# 运行 Go -> C++ 端到端冒烟
make smoke

# Go 单测
make test
```

启动本地模拟交易服务：

```bash
make run
```

另开终端提交订单：

```bash
curl -X POST http://127.0.0.1:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"symbol":"DEMO","side":"BUY","quantity":10,"limit_price":100}'

curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/snapshot
```

清理编译产物和生成信号：

```bash
make clean
```

## 5. 当前风险限制

配置来源：`config/system.json`；Go 控制面当前也使用同一组默认值，但尚未自动读取 JSON 配置文件。

| 限制 | 当前值 | 位置 |
|---|---:|---|
| 初始现金 | 1,000,000.00 | C++ / Go 默认值 |
| 单笔最大数量 | 100 | Go 控制面 |
| 单标的最大绝对仓位 | 1,000 | Go 控制面 |
| 单笔最大名义金额 | 100,000.00 | Go 控制面 |

当前 C++ 模拟撮合会按订单限价直接成交，不建订单簿，不模拟部分成交、手续费、滑点、交易时段、涨跌停、停牌或资金冻结。

## 6. 验收基线

示例数据执行 `make research` 后，Java 回测输出以下字段：

- `rows`：输入行情行数
- `total_return`：从归一化权益 1.0 计算的总收益
- `max_drawdown`：权益曲线最大回撤
- `trades`：信号从非多头切换到多头的次数
- `lookahead=false`：当前收益使用下一交易日收盘价计算

`make smoke` 验证：

1. Go HTTP 健康检查成功；
2. 合法订单被控制面接受；
3. 超过单笔数量限制的订单被 HTTP 400 拒绝；
4. C++ 成交事件被 Go 消费；
5. 快照中的 DEMO 持仓更新为 10。

## 7. 文档入口

- [文档索引](docs/README.md)
- [架构与边界](docs/architecture.md)
- [架构可视化](docs/architecture.html)
- [接口契约](docs/api.md)
- [研究与回测口径](docs/research.md)
- [开发与验证](docs/development.md)
- [生产化路线](docs/roadmap.md)
- [风险边界](docs/risk-boundaries.md)

## 8. 生产化结论

这个仓库目前是“跨语言边界验证 + 本地可运行 MVP”，不是可直接接入资金的交易系统。生产化前至少需要补齐：

- 真实行情和柜台适配器；
- 订单幂等、持久化、恢复和断线重连；
- Protobuf/FlatBuffers 代码生成和版本兼容策略；
- 精确资金、仓位、成交和审计账；
- 手续费、滑点、部分成交和拒单模型；
- 时钟同步、限频、熔断、kill switch 和权限隔离；
- 压测、故障演练、VaR/CVaR 和压力测试；
- 真实环境前的仿真盘、灰度和人工复核。
