package httpapi

import (
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
)

func (s *Server) handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, service.NewError(service.CodeNotImplemented, "document operation is not implemented yet", nil))
}
