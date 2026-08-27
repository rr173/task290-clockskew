# BENZHI 评测说明

基于 Go 实现的芯片时钟树偏斜豁免依赖验证 Web 项目，一款后端服务，完成偏斜测量导入与摘要幂等、豁免前提条件匹配与依赖传播、失效豁免定位与验证快照发布。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/task290 --addr :8080 --db task290.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/task290 --smoke-test
```

`--smoke-test` 会真实创建签核批次、注册 functional/scan 模式与 typ 工艺角、导入功能模式 50ps 与扫描模式 950ps 两条偏斜测量（含幂等去重断言）、构造仅部分模式成立的豁免、复现扫描模式超限失效、撤销失效豁免后复验通过、发布不可变验证快照并封存批次，关闭并重新打开数据库验证重启恢复，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/task290 --smoke-test
```

## HTTP API（前缀 /api）

批次：`POST /api/batches`、`GET /api/batches`、`GET /api/batches/{id}`、`POST /api/batches/{id}/open`、`POST /api/batches/{id}/seal`
基准：`POST|GET /api/batches/{id}/endpoints`、`POST|GET /api/batches/{id}/modes`、`POST|GET /api/batches/{id}/corners`、`POST /api/batches/{id}/corners/lock`
测量：`POST|GET /api/batches/{id}/measurements`、`POST /api/measurements/{id}/exclude`
豁免：`POST|GET /api/batches/{id}/exemptions`、`GET /api/exemptions/{id}`、`POST /api/exemptions/{id}/conditions`、`POST /api/exemptions/{id}/dependencies`、`POST /api/exemptions/{id}/revoke`、`POST /api/exemptions/{id}/confirm`
验证：`POST /api/batches/{id}/validate`、`GET /api/batches/{id}/runs`、`GET /api/batches/{id}/failures`
快照：`POST|GET /api/batches/{id}/snapshots`、`POST /api/snapshots/{id}/publish`、`GET /api/snapshots/{id}`
统计：`GET /api/stats`、`GET /api/health`

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。表：batches、clock_endpoints、modes、corners、batch_corners、skew_measurements、exemptions、exemption_conditions、exemption_dependencies、validation_runs、validation_failures、validation_snapshots、snapshot_items。测量以 `(batch_id, seq)` 与内容摘要 checksum 双重幂等；验证运行记录断点游标支持重启续跑；封存快照固定内容哈希与工艺角，发布后不可变。
