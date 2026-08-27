package httpapi

import (
	"net/http"
)

// statsCollect 返回全库统计。
func (s *Server) statsCollect(w http.ResponseWriter, r *http.Request) error {
	st, err := s.st.Collect()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, st)
	return nil
}
