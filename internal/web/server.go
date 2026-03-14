package web

import (
	"net/http"
	"fmt"

	"github.com/alexr/garden-app/internal/planting"
	"github.com/alexr/garden-app/internal/store"
	"github.com/alexr/garden-app/internal/web/handlers"
)

// AppContext holds the shared services passed from the CLI layer.
type AppContext struct {
	Store    store.Store
	FrostSvc *planting.FrostDateService
	Calc     *planting.Calculator
}

// NewServer builds an http.Server with all routes registered.
func NewServer(ac *AppContext, port int) *http.Server {
	h := handlers.New(ac.Store, ac.FrostSvc, ac.Calc)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
}
