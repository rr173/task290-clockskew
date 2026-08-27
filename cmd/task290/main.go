// Command task290 是芯片时钟树偏斜豁免依赖验证服务的入口：
// 默认启动 HTTP 服务；指定 --smoke-test 时执行端到端自检
// （含数据库关闭重开的持久化与重启恢复验证）。
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task290-clockskew/internal/endpoint"
	"task290-clockskew/internal/exemption"
	"task290-clockskew/internal/httpapi"
	"task290-clockskew/internal/measurement"
	"task290-clockskew/internal/service"
	"task290-clockskew/internal/snapshot"
	"task290-clockskew/internal/store"
	"task290-clockskew/internal/validate"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "task290.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end self check and exit")
	flag.Parse()

	if *smoke {
		if err := runSmoke(*dbPath); err != nil {
			log.Fatalf("smoke-test failed: %v", err)
		}
		fmt.Println("smoke-test OK: validate, revoke, revalidate, snapshot, seal, restart-recovery all passed")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	handler := buildAPI(db)
	log.Printf("task290-clockskew listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// buildAPI 组装全部依赖并返回 HTTP handler。
func buildAPI(db *sql.DB) http.Handler {
	bs := store.NewBatchStore(db)
	rs := store.NewReferenceStore(db)
	ms := store.NewMeasurementStore(db)
	es := store.NewExemptionStore(db)
	vs := store.NewValidationStore(db)
	ss := store.NewSnapshotStore(db)

	epSvc := endpoint.NewService(rs)
	meSvc := measurement.NewService(ms, rs)
	exSvc := exemption.NewService(es, rs)
	vaSvc := validate.NewService(vs, es, ms, rs)
	snSvc := snapshot.NewService(ss, vs, es, rs)
	orch := service.NewOrchestrator(bs, ms, vs, vaSvc, snSvc, exSvc)
	stSvc := service.NewStatsService(bs, rs, ms, es, ss)

	return httpapi.New(orch, epSvc, meSvc, exSvc, vs, snSvc, stSvc)
}

// runSmoke 执行端到端自检：建库 → 注册基准 → 导入测量（含幂等）→ 验证发现失效 →
// 撤销豁免 → 复验通过 → 发布快照 → 封存 → 关闭重开同一数据库验证恢复。
func runSmoke(dbPath string) error {
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return err
	}

	bs := store.NewBatchStore(db)
	rs := store.NewReferenceStore(db)
	ms := store.NewMeasurementStore(db)
	es := store.NewExemptionStore(db)
	vs := store.NewValidationStore(db)
	ss := store.NewSnapshotStore(db)

	epSvc := endpoint.NewService(rs)
	meSvc := measurement.NewService(ms, rs)
	exSvc := exemption.NewService(es, rs)
	vaSvc := validate.NewService(vs, es, ms, rs)
	snSvc := snapshot.NewService(ss, vs, es, rs)
	orch := service.NewOrchestrator(bs, ms, vs, vaSvc, snSvc, exSvc)

	// 1. 创建批次（默认偏斜上限 200ps）。
	batch, err := orch.CreateBatch("signoff-batch-1", 200)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}

	// 2. 注册模式与工艺角，锁定 typ 角。
	fnMode, err := epSvc.RegisterMode(batch.ID, "functional", "功能模式")
	if err != nil {
		return err
	}
	scMode, err := epSvc.RegisterMode(batch.ID, "scan", "扫描测试模式")
	if err != nil {
		return err
	}
	typCorner, err := epSvc.RegisterCorner(batch.ID, "typ", "典型工艺角")
	if err != nil {
		return err
	}
	if err := epSvc.LockBatchCorners(batch.ID, []string{typCorner.ID}); err != nil {
		return err
	}

	// 3. 注册端点（clk1/dom1，物理距离约 141um）。
	epA, err := epSvc.RegisterEndpoint(batch.ID, "EP-A", "clk1", "dom1", 0, 0)
	if err != nil {
		return err
	}
	epB, err := epSvc.RegisterEndpoint(batch.ID, "EP-B", "clk1", "dom1", 100, 100)
	if err != nil {
		return err
	}

	// 4. 导入测量：功能模式 50ps 合规；扫描模式 950ps 超限。
	meRes, err := meSvc.Import(batch.ID, []measurement.Input{
		{EndpointA: epA.ID, EndpointB: epB.ID, ModeID: fnMode.ID, CornerID: typCorner.ID, SkewPS: 50, MeasuredAt: "2026-08-27T08:00:00Z"},
		{EndpointA: epA.ID, EndpointB: epB.ID, ModeID: scMode.ID, CornerID: typCorner.ID, SkewPS: 950, MeasuredAt: "2026-08-27T08:05:00Z"},
	})
	if err != nil {
		return err
	}
	if meRes.Inserted != 2 {
		return fmt.Errorf("expected 2 inserted measurements, got %d", meRes.Inserted)
	}

	// 5. 幂等：重复导入同一条测量应被去重（摘要相同）。
	meRes2, err := meSvc.Import(batch.ID, []measurement.Input{
		{EndpointA: epA.ID, EndpointB: epB.ID, ModeID: fnMode.ID, CornerID: typCorner.ID, SkewPS: 50, MeasuredAt: "2026-08-27T08:00:00Z"},
	})
	if err != nil {
		return err
	}
	if meRes2.Duplicates != 1 {
		return fmt.Errorf("expected 1 duplicate, got %d", meRes2.Duplicates)
	}

	// 6. 创建豁免 EX-1：上限 100ps，条件覆盖 functional+scan（typ 角，距离 ≤ 500um）。
	ex1, err := exSvc.Create(batch.ID, "EX-1", epA.ID, epB.ID, 100)
	if err != nil {
		return err
	}
	if _, err := exSvc.AddCondition(ex1.ID, fnMode.ID, typCorner.ID, 500); err != nil {
		return err
	}
	if _, err := exSvc.AddCondition(ex1.ID, scMode.ID, typCorner.ID, 500); err != nil {
		return err
	}

	// 7. 开启批次并验证：扫描模式 950ps 违反豁免上限 → EX-1 失效。
	if _, err := orch.OpenBatch(batch.ID); err != nil {
		return err
	}
	res1, err := orch.ValidateBatch(batch.ID)
	if err != nil {
		return err
	}
	if res1.FailureCount != 1 {
		return fmt.Errorf("expected 1 failure in run-1, got %d", res1.FailureCount)
	}
	b1, err := orch.GetBatch(batch.ID)
	if err != nil {
		return err
	}
	if b1.Status != "has_failures" {
		return fmt.Errorf("expected batch has_failures, got %s", b1.Status)
	}

	// 8. 撤销失效豁免并复验：无失效 → publishable。
	ex1After, err := orch.RevokeExemption(ex1.ID)
	if err != nil {
		return err
	}
	if ex1After.Status != "revoked" {
		return fmt.Errorf("expected exemption revoked, got %s", ex1After.Status)
	}
	res2, err := orch.ValidateBatch(batch.ID)
	if err != nil {
		return err
	}
	if res2.FailureCount != 0 {
		return fmt.Errorf("expected 0 failures in run-2, got %d", res2.FailureCount)
	}
	b2, err := orch.GetBatch(batch.ID)
	if err != nil {
		return err
	}
	if b2.Status != "publishable" {
		return fmt.Errorf("expected batch publishable, got %s", b2.Status)
	}

	// 9. 创建并发布验证快照（固定内容哈希与工艺角），封存批次。
	snap, err := snSvc.CreateDraft(batch.ID, "snapshot-v1")
	if err != nil {
		return err
	}
	if snap.ContentHash == "" {
		return fmt.Errorf("snapshot content hash must not be empty")
	}
	pub, err := snSvc.Publish(snap.ID)
	if err != nil {
		return err
	}
	if pub.Status != "published" {
		return fmt.Errorf("expected snapshot published, got %s", pub.Status)
	}
	sealed, err := orch.SealBatch(batch.ID)
	if err != nil {
		return err
	}
	if sealed.Status != "sealed" {
		return fmt.Errorf("expected batch sealed, got %s", sealed.Status)
	}

	// 10. 重启恢复：关闭后重新打开同一数据库，验证状态、快照哈希与运行记录完整。
	if err := db.Close(); err != nil {
		return err
	}
	db2, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer db2.Close()

	bs2 := store.NewBatchStore(db2)
	bAfter, err := bs2.Get(batch.ID)
	if err != nil {
		return err
	}
	if bAfter.Status != "sealed" {
		return fmt.Errorf("restart: expected batch sealed, got %s", bAfter.Status)
	}
	ss2 := store.NewSnapshotStore(db2)
	snapAfter, err := ss2.Get(snap.ID)
	if err != nil {
		return err
	}
	if snapAfter.ContentHash != snap.ContentHash {
		return fmt.Errorf("restart: snapshot hash changed")
	}
	if snapAfter.Status != "published" {
		return fmt.Errorf("restart: snapshot not published")
	}
	vs2 := store.NewValidationStore(db2)
	runs, err := vs2.Runs(batch.ID)
	if err != nil {
		return err
	}
	if len(runs) != 2 {
		return fmt.Errorf("expected 2 validation runs after restart, got %d", len(runs))
	}
	return nil
}
