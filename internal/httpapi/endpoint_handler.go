package httpapi

import (
	"net/http"
)

// endpointCreate 注册时钟端点。
func (s *Server) endpointCreate(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name        string  `json:"name"`
		ClockName   string  `json:"clock_name"`
		ClockDomain string  `json:"clock_domain"`
		X           float64 `json:"x"`
		Y           float64 `json:"y"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	e, err := s.ep.RegisterEndpoint(pathID(r), req.Name, req.ClockName, req.ClockDomain, req.X, req.Y)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, e)
	return nil
}

// endpointList 列出批次端点。
func (s *Server) endpointList(w http.ResponseWriter, r *http.Request) error {
	items, err := s.ep.ListEndpoints(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// modeCreate 注册工作模式。
func (s *Server) modeCreate(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	m, err := s.ep.RegisterMode(pathID(r), req.Name, req.Description)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, m)
	return nil
}

// modeList 列出批次模式。
func (s *Server) modeList(w http.ResponseWriter, r *http.Request) error {
	items, err := s.ep.ListModes(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// cornerCreate 注册工艺角。
func (s *Server) cornerCreate(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	c, err := s.ep.RegisterCorner(pathID(r), req.Name, req.Description)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, c)
	return nil
}

// cornerList 列出批次工艺角。
func (s *Server) cornerList(w http.ResponseWriter, r *http.Request) error {
	items, err := s.ep.ListCorners(pathID(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, items)
	return nil
}

// cornerLock 将一组工艺角绑定到批次。
func (s *Server) cornerLock(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		CornerIDs []string `json:"corner_ids"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if err := s.ep.LockBatchCorners(pathID(r), req.CornerIDs); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "locked"})
	return nil
}
