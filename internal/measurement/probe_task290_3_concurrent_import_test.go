package measurement_test

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"task290-clockskew/internal/endpoint"
	"task290-clockskew/internal/exemption"
	"task290-clockskew/internal/measurement"
	"task290-clockskew/internal/model"
	"task290-clockskew/internal/service"
	"task290-clockskew/internal/snapshot"
	"task290-clockskew/internal/store"
	"task290-clockskew/internal/validate"
)

type probeEnv struct {
	db   *sql.DB
	orch *service.Orchestrator
	ep   *endpoint.Service
	me   *measurement.Service
	ex   *exemption.Service
	va   *validate.Service
	vs   *store.ValidationStore
	sn   *snapshot.Service
	bs   *store.BatchStore
	ms   *store.MeasurementStore
	es   *store.ExemptionStore
	ss   *store.SnapshotStore
	rs   *store.ReferenceStore
}

func newProbeEnv(t *testing.T) *probeEnv {
	t.Helper()
	db, err := store.Open(t.TempDir() + "/probe.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
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
	return &probeEnv{
		db: db, orch: orch, ep: epSvc, me: meSvc, ex: exSvc, va: vaSvc,
		vs: vs, sn: snSvc, bs: bs, ms: ms, es: es, ss: ss, rs: rs,
	}
}

type probeBatch struct {
	batch     *model.Batch
	epA       *model.ClockEndpoint
	epB       *model.ClockEndpoint
	fnMode    *model.Mode
	scMode    *model.Mode
	typCorner *model.Corner
}

func (p *probeEnv) setupBatch(t *testing.T) *probeBatch {
	t.Helper()
	batch, err := p.orch.CreateBatch("probe-batch", 200)
	if err != nil {
		t.Fatal(err)
	}
	fnMode, err := p.ep.RegisterMode(batch.ID, "functional", "fn")
	if err != nil {
		t.Fatal(err)
	}
	scMode, err := p.ep.RegisterMode(batch.ID, "scan", "scan")
	if err != nil {
		t.Fatal(err)
	}
	typCorner, err := p.ep.RegisterCorner(batch.ID, "typ", "typ")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.ep.LockBatchCorners(batch.ID, []string{typCorner.ID}); err != nil {
		t.Fatal(err)
	}
	epA, err := p.ep.RegisterEndpoint(batch.ID, "EP-A", "clk1", "dom1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	epB, err := p.ep.RegisterEndpoint(batch.ID, "EP-B", "clk1", "dom1", 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	return &probeBatch{batch: batch, epA: epA, epB: epB, fnMode: fnMode, scMode: scMode, typCorner: typCorner}
}

func (pb *probeBatch) open(t *testing.T, p *probeEnv) {
	t.Helper()
	if _, err := p.orch.OpenBatch(pb.batch.ID); err != nil {
		t.Fatal(err)
	}
}

func (pb *probeBatch) importMeasurement(t *testing.T, p *probeEnv, modeID string, skew int64, at string) {
	t.Helper()
	_, err := p.me.Import(pb.batch.ID, []measurement.Input{{
		EndpointA: pb.epA.ID, EndpointB: pb.epB.ID,
		ModeID: modeID, CornerID: pb.typCorner.ID,
		SkewPS: skew, MeasuredAt: at,
	}})
	if err != nil {
		t.Fatal(err)
	}
}

func (pb *probeBatch) createExemption(t *testing.T, p *probeEnv, name string, maxSkew int64, modes ...*model.Mode) *model.Exemption {
	t.Helper()
	ex, err := p.ex.Create(pb.batch.ID, name, pb.epA.ID, pb.epB.ID, maxSkew)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range modes {
		if _, err := p.ex.AddCondition(ex.ID, m.ID, pb.typCorner.ID, 500); err != nil {
			t.Fatal(err)
		}
	}
	return ex
}

func TestConcurrentImportDistinctMeasurements(t *testing.T) {
	p := newProbeEnv(t)
	pb := p.setupBatch(t)
	pb.open(t, p)

	const workers = 20
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 1; i <= workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			at := fmt.Sprintf("2026-08-27T08:%02d:00Z", n)
			_, err := p.me.Import(pb.batch.ID, []measurement.Input{{
				EndpointA: pb.epA.ID, EndpointB: pb.epB.ID,
				ModeID: pb.fnMode.ID, CornerID: pb.typCorner.ID,
				SkewPS: int64(n), MeasuredAt: at,
			}})
			if err != nil {
				errCh <- fmt.Errorf("import %d: %w", n, err)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}

	items, err := p.me.List(pb.batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != workers {
		t.Fatalf("measurement count=%d want=%d", len(items), workers)
	}
}
