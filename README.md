# task290-clockskew 芯片时钟树偏斜豁免依赖验证服务

基于 REQ-20260826-152 生成。数字后端工程师登记时钟端点、偏斜测量、工作模式与工艺角后，
服务匹配测量与豁免条件、沿依赖边传播前提、定位失效豁免并发布不可变验证快照，
回答「这条原本允许的偏斜豁免在扫描模式下是否仍是真实时序风险」。

## 业务背景

时钟树偏斜（clock tree skew）超过默认约束的路径通常以「偏斜豁免」放行。但豁免不是无条件成立——
它依赖模式（functional/scan）、工艺角（typ/ss/ff）与物理距离等前提。若扫描模式下实际测量
偏斜超限而豁免只声明功能模式成立，则该豁免的前提被证伪，必须撤销并重新签核。

本服务将这条验证链固化：导入测量 → 匹配豁免条件 → 传播依赖 → 定位失效 → 撤销/确认 →
发布验证快照 → 封存批次。

## 核心闭环

1. 创建签核批次（receiving），注册时钟端点（含时钟域与物理坐标）、模式与工艺角，锁定工艺角集合。
2. 批量导入偏斜测量（skew_ps、模式、工艺角），内容摘要幂等去重；端点时钟域未知、工艺角缺失拒绝。
3. 创建豁免（candidate），声明上限偏斜与若干（模式, 工艺角, 物理距离上限）前提条件，可挂依赖边（无环）。
4. 开启批次（pending）并验证：匹配测量、传播依赖、裁决每条豁免 valid/invalid，失效写入运行记录。
5. 批次流转 has_failures / publishable；工程师撤销失效豁免后复验，直至通过。
6. 发布验证快照（固定内容哈希与工艺角，发布后不可变），封存批次（sealed，写操作全拒）。

## 状态机

- 批次：receiving → pending → has_failures | publishable → sealed
- 测量：raw → matched | mode_mismatch | excluded
- 豁免：candidate → valid | invalid → revoked → confirmed
- 快照：draft → published → superseded

## 标准命令

```bash
# 构建
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
# 静态检查
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
# 测试
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
# 端到端自检（建库 → 验证 → 撤销 → 复验 → 快照 → 封存 → 重启恢复）
go run ./cmd/task290 --smoke-test
# 启动服务
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/task290 --addr :8080 --db task290.db
```

## API 一览（前缀 /api）

| 能力 | 入口 | 生产实现 |
| --- | --- | --- |
| 创建批次 | `POST /api/batches` | httpapi → service.Orchestrator.CreateBatch → store.BatchStore |
| 开启批次 | `POST /api/batches/{id}/open` | service.Orchestrator.OpenBatch（receiving→pending） |
| 封存批次 | `POST /api/batches/{id}/seal` | service.Orchestrator.SealBatch（需已发布快照） |
| 注册端点 | `POST /api/batches/{id}/endpoints` | httpapi → endpoint.Service.RegisterEndpoint（时钟域校验） |
| 注册模式/工艺角 | `POST /api/batches/{id}/modes|corners` | endpoint.Service.RegisterMode/RegisterCorner |
| 锁定工艺角 | `POST /api/batches/{id}/corners/lock` | endpoint.Service.LockBatchCorners |
| 导入测量 | `POST /api/batches/{id}/measurements` | httpapi → measurement.Service.Import（幂等摘要） |
| 排除测量 | `POST /api/measurements/{id}/exclude` | measurement.Service.Exclude |
| 创建豁免 | `POST /api/batches/{id}/exemptions` | httpapi → exemption.Service.Create |
| 添加条件 | `POST /api/exemptions/{id}/conditions` | exemption.Service.AddCondition（重复组合拒绝） |
| 添加依赖 | `POST /api/exemptions/{id}/dependencies` | exemption.Service.AddDependency（环检测） |
| 撤销豁免 | `POST /api/exemptions/{id}/revoke` | service.Orchestrator.RevokeExemption（批次回 pending） |
| 确认豁免 | `POST /api/exemptions/{id}/confirm` | exemption.Service.Confirm（仅 valid） |
| 运行验证 | `POST /api/batches/{id}/validate` | httpapi → service.Orchestrator.ValidateBatch → validate.Service |
| 失效清单 | `GET /api/batches/{id}/failures` | store.ValidationStore.LastFailures |
| 快照草稿 | `POST /api/batches/{id}/snapshots` | snapshot.Service.CreateDraft（内容哈希） |
| 发布快照 | `POST /api/snapshots/{id}/publish` | snapshot.Service.Publish（旧快照替代） |
| 统计 | `GET /api/stats` | service.StatsService.Collect |

## 持久化与重启恢复

SQLite（modernc.org/sqlite v1.52.0，CGO 无关，离线可构建）。测量摘要 `sha256(batch|端点对|模式|角|skew|时间)` 唯一，
重复导入幂等跳过；验证运行保存 `cursor_measurement/cursor_exemption` 断点游标，进程重启后从游标续跑；
封存快照绑定内容哈希与工艺角集合，`--smoke-test` 关闭并重开同一数据库验证状态、哈希与运行记录完整恢复。

## 关键不变量

- 测量摘要幂等：同内容重复导入不产生新行；`(batch_id, seq)` 唯一。
- 端点时钟域未知拒绝；测量/条件引用的工艺角缺失拒绝。
- 豁免依赖图无环（创建时 DFS + 验证时拓扑排序双重防御）。
- 快照发布后内容哈希不可变；批次 sealed 后所有写操作被拒。
- 同一批次同一时刻至多一个 running 验证运行（裁决串行）。
