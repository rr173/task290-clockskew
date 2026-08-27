package httpapi

import (
	"net/http"
)

// snapshotCreateDraft 从批次当前验证结论创建草稿快照。
func (s *Server) snapshotCreateDraft(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	snap, err := s.sn.CreateDraft(pathID(r), req.Name)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, snap)
	return nil
}

// snapshotList 列出批次全部快照。
func (s *Server) snapshotList(w http.ResponseWriter, r *http.Request) error {
	items, err := s.sn.List(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// snapshotPublish 发布草稿快照（旧已发布快照自动替代）。
func (s *Server) snapshotPublish(w http.ResponseWriter, r *http.Request) error {
	snap, err := s.sn.Publish(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, snap)
	return nil
}

// snapshotGet 查询快照详情与摘要条目。
func (s *Server) snapshotGet(w http.ResponseWriter, r *http.Request) error {
	snap, items, err := s.sn.Get(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": snap, "items": items})
	return nil
}
