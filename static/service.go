package static

import (
	"net/http"
	"path"
	"strings"
	rrttp "github.com/spiral/roadrunner/http"
	"github.com/spiral/roadrunner/service"
)

// Name contains default service name.
const Name = "static"

// Service serves static files. Potentially convert into middleware?
type Service struct {
	// server configuration (location, forbidden files and etc)
	cfg *Config

	// root is initiated http directory
	root http.Dir

	// let's service stay running
	done chan interface{}
}

// Configure must return configure service and return true if service hasStatus enabled. Must return error in case of
// misconfiguration. Services must not be used without proper configuration pushed first.
func (s *Service) Configure(cfg service.Config, c service.Container) (enabled bool, err error) {
	config := &Config{}
	if err := cfg.Unmarshal(config); err != nil {
		return false, err
	}

	if !config.Enable {
		return false, nil
	}

	if err := config.Valid(); err != nil {
		return false, err
	}

	s.cfg = config
	s.root = http.Dir(s.cfg.Dir)

	// registering as middleware
	if h, ok := c.Get(rrttp.Name); ok >= service.StatusConfigured {
		if h, ok := h.(*rrttp.Service); ok {
			h.Add(s)
		}
	}

	return true, nil
}

// Serve serves the service.
func (s *Service) Serve() error {
	s.done = make(chan interface{})
	<-s.done

	return nil
}

// Stop stops the service.
func (s *Service) Stop() {
	close(s.done)
}

// Handle must return true if request/response pair is handled withing the middleware.
func (s *Service) Handle(w http.ResponseWriter, r *http.Request) bool {
	fPath := r.URL.Path
	if !strings.HasPrefix(fPath, "/") {
		fPath = "/" + fPath
	}
	fPath = path.Clean(fPath)

	if s.cfg.Forbids(fPath) {
		return false
	}

	f, err := s.root.Open(fPath)
	if err != nil {
		return false
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		return false
	}

	// do not Handle directories
	if d.IsDir() {
		return false
	}

	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
	return true
}