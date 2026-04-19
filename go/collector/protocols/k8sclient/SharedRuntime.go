package k8sclient

import (
	"sync"

	"github.com/saichler/l8types/go/ifs"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// sharedRuntimeState holds process-wide singleton state shared by all
// ClientGoCollector instances (one per host-protocol pair).
// Access is guarded by its internal mutex.
type sharedRuntimeState struct {
	mu            sync.Mutex
	cache         *CollectorCache
	restConfig    *rest.Config
	dynamicClient dynamic.Interface
	warmed        map[string]bool
	warmOnce      map[string]*sync.Once
	stopCh        chan struct{}
	connected     bool
	serverStarted bool
}

var shared = &sharedRuntimeState{}

func (s *sharedRuntimeState) init() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache == nil {
		s.cache = NewCollectorCache()
	}
	if s.warmed == nil {
		s.warmed = make(map[string]bool)
	}
	if s.warmOnce == nil {
		s.warmOnce = make(map[string]*sync.Once)
	}
	if s.stopCh == nil {
		s.stopCh = make(chan struct{})
	}
}

// connect establishes the dynamic client once. Subsequent calls reuse the
// existing connection. Returns the rest config and dynamic client.
func (s *sharedRuntimeState) connect(configFn func() (*rest.Config, error)) (*rest.Config, dynamic.Interface, error) {
	s.mu.Lock()
	if s.connected && s.dynamicClient != nil {
		cfg, client := s.restConfig, s.dynamicClient
		s.mu.Unlock()
		return cfg, client, nil
	}
	s.mu.Unlock()

	cfg, err := configFn()
	if err != nil {
		return nil, nil, err
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check: another goroutine may have connected while we were
	// building the client.
	if s.connected && s.dynamicClient != nil {
		return s.restConfig, s.dynamicClient, nil
	}
	s.restConfig = cfg
	s.dynamicClient = client
	s.connected = true
	return cfg, client, nil
}

// ensureAdmissionServer starts the admission HTTPS server exactly once.
// Returns true if it was already started (no action needed).
func (s *sharedRuntimeState) ensureAdmissionServer(startFn func() error) error {
	s.mu.Lock()
	if s.serverStarted {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	err := startFn()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.serverStarted = true
	s.mu.Unlock()
	return nil
}

// onceForKey returns a sync.Once for the given warm key, creating it if needed.
func (s *sharedRuntimeState) onceForKey(key string) *sync.Once {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.warmOnce == nil {
		s.warmOnce = make(map[string]*sync.Once)
	}
	once, ok := s.warmOnce[key]
	if !ok {
		once = &sync.Once{}
		s.warmOnce[key] = once
	}
	return once
}

// markWarmed records that the informer for key is running.
func (s *sharedRuntimeState) markWarmed(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.warmed[key] = true
}

// isWarmed checks whether an informer is already running for key.
func (s *sharedRuntimeState) isWarmed(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.warmed[key]
}

// disconnect tears down the shared connection. All informers are stopped.
func (s *sharedRuntimeState) disconnect(logger ifs.ILogger) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
			// already closed
		default:
			close(s.stopCh)
			if logger != nil {
				logger.Info("shared runtime: stopped all informers")
			}
		}
	}
	s.connected = false
	s.dynamicClient = nil
	s.restConfig = nil
	s.warmed = nil
	s.warmOnce = nil
	s.stopCh = nil
	s.serverStarted = false
	if logger != nil {
		logger.Info("shared runtime: disconnected")
	}
}
