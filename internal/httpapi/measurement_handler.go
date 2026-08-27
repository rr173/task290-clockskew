package httpapi

import (
	"net/http"

	"task290-clockskew/internal/measurement"
)

// measurementImport 批量导入测量（幂等）。
func (s *Server) measurementImport(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Items []measurement.Input `json:"items"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	res, err := s.me.Import(pathID(r), req.Items)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, res)
	return nil
}

// measurementList 列出批次测量。
func (s *Server) measurementList(w http.ResponseWriter, r *http.Request) error {
	items, err := s.me.List(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// measurementExclude 将测量标记为排除。
func (s *Server) measurementExclude(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if err := s.me.Exclude(pathID(r), req.Reason); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "excluded"})
	return nil
}
