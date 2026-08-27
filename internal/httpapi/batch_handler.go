package httpapi

import (
	"net/http"
)

// batchCreate 创建批次（receiving）。
func (s *Server) batchCreate(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name             string `json:"name"`
		DefaultMaxSkewPS int64  `json:"default_max_skew_ps"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	b, err := s.orch.CreateBatch(req.Name, req.DefaultMaxSkewPS)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, b)
	return nil
}

// batchList 列出全部批次。
func (s *Server) batchList(w http.ResponseWriter, r *http.Request) error {
	bs, err := s.orch.ListBatches()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, bs)
	return nil
}

// batchGet 查询批次详情。
func (s *Server) batchGet(w http.ResponseWriter, r *http.Request) error {
	b, err := s.orch.GetBatch(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

// batchOpen 开启批次：receiving → pending。
func (s *Server) batchOpen(w http.ResponseWriter, r *http.Request) error {
	b, err := s.orch.OpenBatch(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

// batchSeal 封存批次：publishable + 已发布快照 → sealed。
func (s *Server) batchSeal(w http.ResponseWriter, r *http.Request) error {
	b, err := s.orch.SealBatch(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}
