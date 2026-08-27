package httpapi

import (
	"net/http"
)

// validate 触发（或续跑）一轮验证，并自动流转批次状态。
func (s *Server) validate(w http.ResponseWriter, r *http.Request) error {
	res, err := s.orch.ValidateBatch(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, res)
	return nil
}

// validationRuns 列出批次验证运行历史。
func (s *Server) validationRuns(w http.ResponseWriter, r *http.Request) error {
	items, err := s.vs.Runs(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// validationFailures 返回批次最近一次完成运行的失效清单。
func (s *Server) validationFailures(w http.ResponseWriter, r *http.Request) error {
	items, err := s.vs.LastFailures(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}
