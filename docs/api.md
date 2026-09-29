# 接口契约

## 1. HTTP API

默认监听地址：`127.0.0.1:8080`

### `GET /health`

用途：检查 Go 控制面是否能提供 HTTP 服务。

响应：

```json
{
  "status": "ok",
  "control_plane": "go",
  "execution_core": "cpp"
}
```

### `POST /orders`

请求头：

```text
Content-Type: application/json
```

请求体：

```json
{
  "order_id": "optional-client-id",
  "symbol": "DEMO",
  "side": "BUY",
  "quantity": 10,
  "limit_price": 100.0
}
```

字段：

| 字段 | 类型 | 必填 | 规则 |
|---|---|---:|---|
| `order_id` | string | 否 | 为空时由 Go 生成 `go-N` |
| `symbol` | string | 是 | 非空，长度不超过 32 |
| `side` | string | 是 | `BUY` 或 `SELL`，输入会转大写 |
| `quantity` | int64 | 是 | `1 <= quantity <= 100` |
| `limit_price` | number | 是 | 大于 0，且 `quantity * limit_price <= 100000` |

成功响应：HTTP `202`

```json
{
  "order_id": "smoke-1",
  "status": "accepted_by_control_plane"
}
```

注意：`202` 只表示控制面已经把订单写入执行核心 stdin，不表示成交已经完成。应通过 `/snapshot` 或未来的事件订阅接口确认最终状态。

拒绝响应：HTTP `400`

```json
{
  "error": "quantity must be in [1,100]"
}
```

执行核心不可用：HTTP `503`

```json
{
  "error": "execution core unavailable"
}
```

### `GET /snapshot`

当前返回内存快照：

```json
{
  "cash": 999000,
  "positions": {"DEMO": 10},
  "events": [
    {"type": "accepted", "order_id": "smoke-1"},
    {"type": "fill", "order_id": "smoke-1", "symbol": "DEMO"}
  ]
}
```

`events` 只保留最近 100 条；该接口不保证跨重启连续性。

## 2. Go -> C++ NDJSON

当前每条消息占一行。

### 订单消息

```json
{"type":"order","order_id":"smoke-1","symbol":"DEMO","side":"BUY","quantity":10,"limit_price":100}
```

### 查询消息

```json
{"type":"query"}
```

### 接受事件

```json
{"type":"accepted","order_id":"smoke-1","ts_ns":1790639225365199000}
```

### 成交事件

```json
{"type":"fill","order_id":"smoke-1","symbol":"DEMO","side":"BUY","filled_quantity":10,"fill_price":100.00,"position":10,"cash":999000.00,"ts_ns":1790639225365216000}
```

### 拒绝事件

```json
{"type":"reject","order_id":"smoke-1","reason":"invalid_order","ts_ns":1790639225365216000}
```

## 3. Protobuf 演进草案

文件：`proto/events.proto`

当前定义了 `Order` 和 `ExecutionEvent` 的基础字段。生产版本需要补充：

```text
event_id
sequence
account_id
venue
instrument_id
order_type
time_in_force
status
reject_code
fees
liquidity_flag
received_timestamp_ns
processed_timestamp_ns
```

建议规则：

- 不复用已经发布的 field number；
- 删除字段使用 `reserved`；
- 所有事件带单调递增 sequence；
- 消费方按 `event_id` 去重；
- 重放必须保证同样的顺序和状态转移；
- 外部交易所时间和本机接收时间分开保存。

## 4. 错误语义

| 层 | 错误 | 当前行为 | 生产目标 |
|---|---|---|---|
| HTTP | JSON 格式错误 | `400` | 增加错误码和 request_id |
| HTTP | 风控拒绝 | `400` | 区分参数错误、限额、熔断、权限 |
| IPC | C++ 子进程不可用 | `503` | 熔断、恢复、未确认订单查询 |
| C++ | 基础订单字段错误 | 输出 `reject` | 统一拒单码和可审计原文 |
| 业务 | 重复订单 | 当前未检测 | `order_id` 幂等和最终状态查询 |
