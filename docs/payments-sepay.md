# SePay 支付运营手册

覆盖 SePay 银行转账支付的配置、回调注册、密钥轮换、日常对账与常见问题。读懂本手册前，建议先查看 *设计文档* 中 SePay 各决策（D1–D11）与 `setting/payment_sepay.go`、`model/sepay.go` 的实现（唯一约束、幂等与派生过期）。

## 开通前合规确认

SePay 接入前须在后台管理面板 `系统设置 → 运营设置 → 支付合规` 中确认合规承诺（`payment_setting.compliance_*`）。未完成前，`POST /api/user/sepay/pay` 与 `/api/subscription/sepay/pay` 会因 `operation_setting.IsPaymentComplianceConfirmed() == false` 而被拒。合规项不可通过通用 `UpdateOption` 通道改写。

## 填写 SePay 设置

进入后台 `系统设置 → 支付设置 → SePay`。必填字段如下（任意缺项均视为「未配置可用」—— `setting.IsSePayConfigured() == false` 时订单创建被拒）：

| 键 | 含义 | 约束 |
|----|------|------|
| 是否启用 `SePayEnabled` | 总开关 | `true` 才能创建 SePay 订单 |
| 收款账号 `SePayBankAccountNumber` | 接收转账的银行卡号 | 去空格后非空 |
| 银行编码 `SePayBankCode` | SePay 二维码中使用的银行代号 | 去空格后非空；形式由 SePay QR 规范为准，operator 自行与 SePay 核对 |
| 户名 `SePayAccountHolder` | 收款人姓名 | 去空格后非空 |
| Webhook API 密钥 `SePayWebhookApiKey` | 与 SePay 控制台一致的回调密钥 | 去空格后非空；**在接口返回中对非管理员不可见**（`GetOptions` 对以 `Key` 结尾的选项默认脱敏），且**被视为密钥** |
| 最低充值 `SePayMinTopUp` | 单笔订单最低 USD 金额 | 默认 1；服务端会在创建时校验极小值并拒绝生成 0 Dong 订单 |
| 过期窗口 `SePayOrderExpiryMinutes` | 待付订单可支付时长（分钟） | 1–10080，默认 30；超范围的提交会被 `ValidateSePayOrderExpiryMinutes` 拒绝 |

`price` 复用既有通用支付设置中的 **Dong/美元**（默认 1000）。`$10` 的充值订单应付为 10,000 东（`amount × price × topup_group_ratio × discount`，以 `decimal` 运算并保存在既有 `top_ups.money` 列的整数 Dong 上；`common.WalletQuotaFromDecimalStrict` 覆盖换算）。

`GetTopUpInfo` 不再返回任何退休网关（Epay、Stripe、Creem、Waffo、Waffo Pancake）的开关、商品或支付方式字段，仅 expose SePay 的可用性与共享计价字段。

## 向 SePay 注册回调

1. 在 SePay 控制台绑定收款账号的 Webhook URL：
   ```
   https://<host>/api/sepay/webhook
   ```
2. `Apikey` 填与后台一致的 `SePayWebhookApiKey`。
3. SePay 通过 `Authorization: Apikey <key>` 调用该地址（大写 `A`），以 `crypto/subtle.ConstantTimeCompare` 校验；缺失、错误 scheme（如 `Bearer` 或小写 `apikey`）、错误值或 `SePayEnabled == false`、`SePayWebhookApiKey` 为空时一律返回 `401`，否则按下面分支返回 `200`。
4. 该地址必须可从公网访问且走 **HTTPS**；否则 SePay 无法送达，支付不会被结算。

## 轮换 API 密钥

1. 在后台 SePay 设置中填入新的 `SePayWebhookApiKey` 并保存。
2. 同时到 SePay 控制台将 `Apikey` 改为同一新值。
3. 若保存时留空，则**保留原值不替换**（`model.UpdateOption("SePayWebhookApiKey","")` 不写库），避免在仅编辑另一个 SePay 字段时意外清空回调密钥。

## 处理「未匹配 / 少付 / 多付」转账

Webhook 的文本字段 `content` 会先正规化为大写 `A-Z0-9`，再用正则提取形如 `SP...` / `SS...` 的候选 memo，逐一按 trade_no 精确等值匹配（**不用 `LIKE` 扫描**）；前缀 `SP` 走 `top_ups`，`SS` 走 `subscription_orders`。以下均为 **200 成功**（仅错误分支写 `warning/error` 日志，携带 SePay 的 `id` 与 `referenceCode`；只有鉴权失败与真实 5xx 返回非 2xx，SePay 仅在非 2xx 时重试）：

* **已到账**：满足金额且未过期的待付订单 → 标记 `success` 并计入用户余额（`RechargeSePay` / `CompleteSubscriptionOrder` 内事务加 `lockForUpdate`，行级幂等；多付则按订单所定礼包全额入账并对 surplus 打一条 `SysError` 日志）。
* **已到账重复投递**：相同 SePay 事件再次投递 → `alreadyDone=true`，不重复入账。
* **少付**：转账金额 `< 冻结应付 Dong` → 留待付（之后按过期规则飘红），唯有一条 `warning`。
* **多付**：转账金额 `> 冻结应付 Dong` → 按**订单金额**入账（不按转账金额），对 `surplus = transfer - payable` 打一条 `SysError` 现场记录，余款不自动退也不自动补。
* **往外转账（outgoing transfer）**、`无匹配 memo`、`已过期`（创建时间 + 配置窗口 `< now`，即使清扫尚未将状态物化为 `expired` 也不入账）、`一次 content 命中两条待付订单`（模糊匹配）→ 均不入账并记日志，operator 再用下面描述的「管理员补单」手段人工处理。

建议每日到 `Logs → 操作日志` 中按关键词搜索 `SePay` 与 `referenceCode` 巡检异常项。

## 清扫过期订单

`SePayOrderExpiryUnix(createTime) = create_time + 配置分钟数×60`。Webhook 在结算前**直接用派生时间判断是否已过 deadline**，因此无需等待清扫也会拒付过期订单。系统任务 runner 另有一道**仅对 SePay** 的批量过期清扫，将过期待付行批量标记为 `expired` 并更新 `complete_time`，方便列表页与统计正确展示。**退休网关的待付单永远不会被该清扫标记为过期**。

## 处理退休网关（Epay / Stripe / Creem / Waffo / Waffo Pancake）遗留待付单

切换至 SePay 后，原有五家网关的创建与回调端点已下线（调用返回 404），对应的 SDK 与设置模块也已移除，且其待付单不会被新的过期清扫自动消化。operator 应在切前导出待付单清单，核对线下实际到账后，通过后台「管理员补单」人工结算：

* `POST /api/admin/topup/complete`（由 `controller.AdminCompleteTopUp` 暴露，仍沿用 `model.ManualCompleteTopUp`）以传入的 `trade_no` 为准，在事务内 `lockForUpdate`、校验待付状态、通过 `common.WalletQuotaFromDecimalStrict` 换算应充额度（含 Stripe 分组换算的子分支），并以 `creditTopUpQuota` 的条件 `UPDATE` 原子越过余额上限；成功后写入一条由 `admin` 触发的充值日志；重复补单为幂等（已成功则直接返回）。
* 回滚：由于本次变更对 `top_ups` / `subscription_orders` **无任何列或约束变更**（trade_no 唯一索引不变；GORM struct tag 与 `HEAD` 完全一致 — 可用 `git diff HEAD -- model/topup.go model/subscription.go` 直接验证），旧版本代码可直接重新部署，期间产生的 SePay 订单仍以 `ManualCompleteTopUp` 在旧版上线后人工入账。

## 安全说明

* `SePayWebhookApiKey` 是**密钥**：永远不对非管理员可见；UI 中已存值以掩码展示；建议定期轮换并与银行收款账号的访问权限一同审计。
* 高风险端点已走 `middleware.CriticalRateLimit()`（订单创建）与 `anonymousRequestBodyLimit`（Webhook）；Webhook 亦受匿名限流保护。
