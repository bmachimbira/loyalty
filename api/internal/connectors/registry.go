package connectors

import (
	"fmt"
	"sync"
	"time"
)

// Registry manages a collection of connectors with circuit breakers
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]*ConnectorWrapper
}

// ConnectorWrapper wraps a connector with a circuit breaker
type ConnectorWrapper struct {
	Connector      Connector
	CircuitBreaker *CircuitBreaker
}

// NewRegistry creates a new connector registry
func NewRegistry() *Registry {
	return &Registry{
		connectors: make(map[string]*ConnectorWrapper),
	}
}

// Register registers a connector with a circuit breaker
func (r *Registry) Register(connector Connector) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Create circuit breaker for this connector
	// 5 failures before opening, 60 second timeout
	cb := NewCircuitBreaker(5, 60*time.Second)

	r.connectors[connector.Name()] = &ConnectorWrapper{
		Connector:      connector,
		CircuitBreaker: cb,
	}
}

// Get retrieves a connector by name
func (r *Registry) Get(name string) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wrapper, ok := r.connectors[name]
	if !ok {
		return nil, false
	}

	return wrapper.Connector, true
}

// GetWithCircuitBreaker retrieves a connector wrapper with circuit breaker
func (r *Registry) GetWithCircuitBreaker(name string) (*ConnectorWrapper, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wrapper, ok := r.connectors[name]
	return wrapper, ok
}

// List returns all registered connector names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.connectors))
	for name := range r.connectors {
		names = append(names, name)
	}

	return names
}

// Unregister removes a connector from the registry
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.connectors, name)
}

// ResetCircuitBreaker manually resets a connector's circuit breaker
func (r *Registry) ResetCircuitBreaker(name string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wrapper, ok := r.connectors[name]
	if !ok {
		return fmt.Errorf("connector %s not found", name)
	}

	wrapper.CircuitBreaker.Reset()
	return nil
}

// GetCircuitBreakerState returns the state of a connector's circuit breaker
func (r *Registry) GetCircuitBreakerState(name string) (CircuitState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wrapper, ok := r.connectors[name]
	if !ok {
		return "", fmt.Errorf("connector %s not found", name)
	}

	return wrapper.CircuitBreaker.State(), nil
}

// Global registry instance
var globalRegistry = NewRegistry()

// GlobalRegistry returns the global connector registry
func GlobalRegistry() *Registry {
	return globalRegistry
}

// RegisterGlobal registers a connector in the global registry
func RegisterGlobal(connector Connector) {
	globalRegistry.Register(connector)
}

// GetGlobal retrieves a connector from the global registry
func GetGlobal(name string) (Connector, bool) {
	return globalRegistry.Get(name)
}

// ListGlobal returns all connector names from the global registry
func ListGlobal() []string {
	return globalRegistry.List()
}
