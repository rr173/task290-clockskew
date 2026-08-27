// Package httpapi 暴露 HTTP API（前缀 /api），编排 store/service 各层。
package httpapi

import (
	"log"
	"net/http"

	"task290-clockskew/internal/endpoint"
	"task290-clockskew/internal/exemption"
	"task290-clockskew/internal/measurement"
	"task290-clockskew/internal/service"
	"task290-clockskew/internal/snapshot"
	"task290-clockskew/internal/store"
)

// Server 持有全部依赖并挂载各业务 handler 方法。
type Server struct {
	orch *service.Orchestrator
	ep   *endpoint.Service
	me   *measurement.Service
	ex   *exemption.Service
	vs   *store.ValidationStore
	sn   *snapshot.Service
	st   *service.StatsService
}

// New 组装依赖并返回带全部路由的 http.Handler。
func New(orch *service.Orchestrator, ep *endpoint.Service, me *measurement.Service,
	ex *exemption.Service, vs *store.ValidationStore, sn *snapshot.Service,
	st *service.StatsService) http.Handler {
	sv := &Server{orch: orch, ep: ep, me: me, ex: ex, vs: vs, sn: sn, st: st}
	mux := http.NewServeMux()

	// 批次生命周期。
	mux.HandleFunc("POST /api/batches", sv.handle(sv.batchCreate))
	mux.HandleFunc("GET /api/batches", sv.handle(sv.batchList))
	mux.HandleFunc("GET /api/batches/{id}", sv.handle(sv.batchGet))
	mux.HandleFunc("POST /api/batches/{id}/open", sv.handle(sv.batchOpen))
	mux.HandleFunc("POST /api/batches/{id}/seal", sv.handle(sv.batchSeal))

	// 基准数据：端点/模式/工艺角。
	mux.HandleFunc("POST /api/batches/{id}/endpoints", sv.handle(sv.endpointCreate))
	mux.HandleFunc("GET /api/batches/{id}/endpoints", sv.handle(sv.endpointList))
	mux.HandleFunc("POST /api/batches/{id}/modes", sv.handle(sv.modeCreate))
	mux.HandleFunc("GET /api/batches/{id}/modes", sv.handle(sv.modeList))
	mux.HandleFunc("POST /api/batches/{id}/corners", sv.handle(sv.cornerCreate))
	mux.HandleFunc("GET /api/batches/{id}/corners", sv.handle(sv.cornerList))
	mux.HandleFunc("POST /api/batches/{id}/corners/lock", sv.handle(sv.cornerLock))

	// 偏斜测量。
	mux.HandleFunc("POST /api/batches/{id}/measurements", sv.handle(sv.measurementImport))
	mux.HandleFunc("GET /api/batches/{id}/measurements", sv.handle(sv.measurementList))
	mux.HandleFunc("POST /api/measurements/{id}/exclude", sv.handle(sv.measurementExclude))

	// 豁免规则。
	mux.HandleFunc("POST /api/batches/{id}/exemptions", sv.handle(sv.exemptionCreate))
	mux.HandleFunc("GET /api/batches/{id}/exemptions", sv.handle(sv.exemptionList))
	mux.HandleFunc("GET /api/exemptions/{id}", sv.handle(sv.exemptionGet))
	mux.HandleFunc("POST /api/exemptions/{id}/conditions", sv.handle(sv.exemptionAddCondition))
	mux.HandleFunc("POST /api/exemptions/{id}/dependencies", sv.handle(sv.exemptionAddDependency))
	mux.HandleFunc("POST /api/exemptions/{id}/revoke", sv.handle(sv.exemptionRevoke))
	mux.HandleFunc("POST /api/exemptions/{id}/confirm", sv.handle(sv.exemptionConfirm))

	// 验证。
	mux.HandleFunc("POST /api/batches/{id}/validate", sv.handle(sv.validate))
	mux.HandleFunc("GET /api/batches/{id}/runs", sv.handle(sv.validationRuns))
	mux.HandleFunc("GET /api/batches/{id}/failures", sv.handle(sv.validationFailures))

	// 快照。
	mux.HandleFunc("POST /api/batches/{id}/snapshots", sv.handle(sv.snapshotCreateDraft))
	mux.HandleFunc("GET /api/batches/{id}/snapshots", sv.handle(sv.snapshotList))
	mux.HandleFunc("POST /api/snapshots/{id}/publish", sv.handle(sv.snapshotPublish))
	mux.HandleFunc("GET /api/snapshots/{id}", sv.handle(sv.snapshotGet))

	// 统计与健康。
	mux.HandleFunc("GET /api/stats", sv.handle(sv.statsCollect))
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return recoverMiddleware(logMiddleware(mux))
}

// handle 将 handler 函数包装为带错误响应的标准签名。
func (s *Server) handle(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			writeError(w, err)
		}
	}
}

// logMiddleware 记录每个请求的方法与路径。
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// recoverMiddleware 捕获 panic 并返回 500，保证服务不因单请求崩溃。
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal panic"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
