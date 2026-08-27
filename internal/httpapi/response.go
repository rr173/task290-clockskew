package httpapi

import (
	"encoding/json"
	"net/http"

	"task290-clockskew/internal/model"
)

// writeJSON 统一 JSON 输出。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 将领域错误映射为 HTTP 状态码并输出。
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch err {
	case model.ErrBatchNotFound, model.ErrEndpointNotFound, model.ErrModeNotFound,
		model.ErrCornerNotFound, model.ErrExemptionNotFound, model.ErrRunNotFound,
		model.ErrSnapshotNotFound:
		status = http.StatusNotFound
	case model.ErrSealedBatchModify, model.ErrSnapshotFrozen, model.ErrBatchAlreadySealed,
		model.ErrDuplicateMeasurement, model.ErrConditionConflict, model.ErrRevokedExemption:
		status = http.StatusConflict
	case model.ErrEndpointDomainKnown, model.ErrCornerMissing, model.ErrRuleCycle,
		model.ErrBatchNotOpen, model.ErrSameEndpointPair, model.ErrInvalidSkew,
		model.ErrMissingDefaultSkew:
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON 解析请求体并返回解析错误。
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

// pathID 从请求路径取 {id} 参数。
func pathID(r *http.Request) string {
	return r.PathValue("id")
}
