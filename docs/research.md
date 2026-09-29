# 研究与回测口径

## 1. 输入数据

示例文件：`data/prices.csv`

```text
date,symbol,close
2026-01-02,DEMO,100.00
```

当前任务只读取三列：

- `date`：日期字符串，要求输入已经按时间升序排列；
- `symbol`：标的；
- `close`：收盘价。

当前实现没有校验日期排序，也没有按 symbol 分组。多标的或乱序输入不能直接视为正确结果。

## 2. Scala 因子任务

入口：`FactorJob <prices.csv> <signals.csv>`

每个交易日计算：

```text
SMA5  = 最近 5 个 close 的平均值
SMA10 = 最近 10 个 close 的平均值
score = (SMA5 - SMA10) / SMA10
```

信号：

```text
long window < 10  -> 0
SMA5 > SMA10      -> 1
否则              -> -1
```

输出格式：

```text
date,symbol,signal,score
2026-01-15,DEMO,1,0.01234567
```

这里的 `signal` 只是研究信号，不是订单。它没有考虑流动性、停牌、交易成本、持仓限制或调仓容量。

## 3. Java 回测

入口：`Backtest <prices.csv> <signals.csv>`

对每个 T 日：

1. 读取 T 日的 signal；
2. 只有 signal 大于 0 才持有多头；
3. 使用 T+1 收盘价计算当日策略收益；
4. 更新归一化权益；
5. 计算峰值和最大回撤。

收益公式：

```text
if signal(T) > 0:
    daily_return(T) = close(T+1) / close(T) - 1
else:
    daily_return(T) = 0
```

这是为了避免用未来数据生成 T 日信号。最后一个价格点没有 T+1，因此不会被计算收益。

输出示例：

```json
{
  "language":"java",
  "job":"backtest",
  "rows":20,
  "total_return":0.06798867,
  "max_drawdown":-0.01289134,
  "trades":1,
  "lookahead":false
}
```

## 4. 当前结果的正确解释

- `total_return` 是示例 CSV 和理想成交假设下的工程结果；
- `max_drawdown` 只来自当前收盘价序列；
- `trades` 统计信号进入多头的次数，不等于真实订单数量；
- 没有手续费、滑点、部分成交、延迟、借券、资金成本和涨跌停约束；
- 不能使用该结果推断任何具体金融产品的未来收益。

## 5. 下一步研究验收

在扩展策略前，优先补齐以下实验框架：

1. 按 symbol 分组、严格日期排序和缺失值校验；
2. 手续费、滑点、最小变动价位和成交延迟；
3. 样本内、验证集、样本外和 walk-forward；
4. 年化收益、波动率、Sharpe、Sortino、最大回撤、Calmar；
5. 分层因子回测、IC、IR 和多空 spread；
6. 市场状态分段：趋势、震荡、高波动、流动性收缩；
7. 极端情景和容量测试；
8. 结果文件带策略版本、数据版本、参数 hash 和运行时间。
