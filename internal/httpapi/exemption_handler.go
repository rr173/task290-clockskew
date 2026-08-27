package httpapi

import (
	"net/http"
)

// exemptionCreate 创建豁免（candidate）。
func (s *Server) exemptionCreate(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name         string `json:"name"`
		FromEndpoint string `json:"from_endpoint"`
		ToEndpoint   string `json:"to_endpoint"`
		MaxSkewPS    int64  `json:"max_skew_ps"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	ex, err := s.ex.Create(pathID(r), req.Name, req.FromEndpoint, req.ToEndpoint, req.MaxSkewPS)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, ex)
	return nil
}

// exemptionList 列出批次豁免。
func (s *Server) exemptionList(w http.ResponseWriter, r *http.Request) error {
	items, err := s.ex.List(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// exemptionGet 查询豁免详情。
func (s *Server) exemptionGet(w http.ResponseWriter, r *http.Request) error {
	ex, err := s.ex.Get(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ex)
	return nil
}

// exemptionAddCondition 为豁免添加前提条件。
func (s *Server) exemptionAddCondition(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		ModeID        string  `json:"mode_id"`
		CornerID      string  `json:"corner_id"`
		MaxDistanceUM float64 `json:"max_distance_um"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	cond, err := s.ex.AddCondition(pathID(r), req.ModeID, req.CornerID, req.MaxDistanceUM)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, cond)
	return nil
}

// exemptionAddDependency 添加豁免依赖边（含环检测）。
func (s *Server) exemptionAddDependency(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		DependsOn string `json:"depends_on"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if err := s.ex.AddDependency(pathID(r), req.DependsOn); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "linked"})
	return nil
}

// exemptionRevoke 撤销豁免（经编排器回退批次状态并清理测量派生状态）。
func (s *Server) exemptionRevoke(w http.ResponseWriter, r *http.Request) error {
	ex, err := s.orch.RevokeExemption(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ex)
	return nil
}

// exemptionConfirm 人工确认豁免（仅 valid）。
func (s *Server) exemptionConfirm(w http.ResponseWriter, r *http.Request) error {
	if err := s.ex.Confirm(pathID(r)); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
	return nil
}
