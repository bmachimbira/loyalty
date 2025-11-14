package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds all application metrics
// This is a simple in-memory implementation
// For production with Prometheus, replace with prometheus.Counter, prometheus.Histogram, etc.
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal     *Counter
	HTTPRequestDuration   *Histogram
	HTTPActiveRequests    *Gauge
	HTTPErrorsTotal       *Counter

	// Business metrics
	EventsProcessedTotal  *Counter
	RulesEvaluatedTotal   *Counter
	RewardsIssuedTotal    *Counter
	RewardsRedeemedTotal  *Counter
	BudgetUtilization     *GaugeVec
	ExternalAPILatency    *Histogram
	CircuitBreakerState   *GaugeVec

	// Database metrics
	DBConnectionsActive   *Gauge
	DBConnectionsIdle     *Gauge
	DBQueryDuration       *Histogram
}

// Counter is a simple counter metric
type Counter struct {
	value int64
}

// Inc increments the counter by 1
func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

// Add adds the given value to the counter
func (c *Counter) Add(val int64) {
	atomic.AddInt64(&c.value, val)
}

// Get returns the current value
func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.value)
}

// Gauge is a metric that can go up and down
type Gauge struct {
	value int64
}

// Set sets the gauge to the given value
func (g *Gauge) Set(val int64) {
	atomic.StoreInt64(&g.value, val)
}

// Inc increments the gauge by 1
func (g *Gauge) Inc() {
	atomic.AddInt64(&g.value, 1)
}

// Dec decrements the gauge by 1
func (g *Gauge) Dec() {
	atomic.AddInt64(&g.value, -1)
}

// Add adds the given value to the gauge
func (g *Gauge) Add(val int64) {
	atomic.AddInt64(&g.value, val)
}

// Get returns the current value
func (g *Gauge) Get() int64 {
	return atomic.LoadInt64(&g.value)
}

// GaugeVec is a collection of gauges with labels
type GaugeVec struct {
	gauges map[string]*Gauge
	mu     sync.RWMutex
}

// NewGaugeVec creates a new GaugeVec
func NewGaugeVec() *GaugeVec {
	return &GaugeVec{
		gauges: make(map[string]*Gauge),
	}
}

// WithLabels returns a gauge for the given label values
func (gv *GaugeVec) WithLabels(labels ...string) *Gauge {
	key := joinLabels(labels...)

	gv.mu.RLock()
	gauge, exists := gv.gauges[key]
	gv.mu.RUnlock()

	if exists {
		return gauge
	}

	gv.mu.Lock()
	defer gv.mu.Unlock()

	// Double-check in case another goroutine created it
	if gauge, exists := gv.gauges[key]; exists {
		return gauge
	}

	gauge = &Gauge{}
	gv.gauges[key] = gauge
	return gauge
}

// Histogram tracks the distribution of values
type Histogram struct {
	observations []time.Duration
	mu           sync.RWMutex
}

// Observe records a new observation
func (h *Histogram) Observe(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.observations = append(h.observations, d)

	// Keep only last 1000 observations to prevent memory issues
	if len(h.observations) > 1000 {
		h.observations = h.observations[len(h.observations)-1000:]
	}
}

// Get returns all observations
func (h *Histogram) Get() []time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]time.Duration, len(h.observations))
	copy(result, h.observations)
	return result
}

// joinLabels joins label values into a single key
func joinLabels(labels ...string) string {
	result := ""
	for i, label := range labels {
		if i > 0 {
			result += ":"
		}
		result += label
	}
	return result
}

var (
	globalMetrics *Metrics
	once          sync.Once
)

// Init initializes the global metrics instance
func Init() *Metrics {
	once.Do(func() {
		globalMetrics = &Metrics{
			// HTTP metrics
			HTTPRequestsTotal:     &Counter{},
			HTTPRequestDuration:   &Histogram{observations: make([]time.Duration, 0, 1000)},
			HTTPActiveRequests:    &Gauge{},
			HTTPErrorsTotal:       &Counter{},

			// Business metrics
			EventsProcessedTotal:  &Counter{},
			RulesEvaluatedTotal:   &Counter{},
			RewardsIssuedTotal:    &Counter{},
			RewardsRedeemedTotal:  &Counter{},
			BudgetUtilization:     NewGaugeVec(),
			ExternalAPILatency:    &Histogram{observations: make([]time.Duration, 0, 1000)},
			CircuitBreakerState:   NewGaugeVec(),

			// Database metrics
			DBConnectionsActive:   &Gauge{},
			DBConnectionsIdle:     &Gauge{},
			DBQueryDuration:       &Histogram{observations: make([]time.Duration, 0, 1000)},
		}
	})
	return globalMetrics
}

// Get returns the global metrics instance
func Get() *Metrics {
	if globalMetrics == nil {
		return Init()
	}
	return globalMetrics
}

// RecordEvent records an event being processed
func RecordEvent() {
	Get().EventsProcessedTotal.Inc()
}

// RecordRuleEvaluation records a rule being evaluated
func RecordRuleEvaluation() {
	Get().RulesEvaluatedTotal.Inc()
}

// RecordRewardIssued records a reward being issued
func RecordRewardIssued() {
	Get().RewardsIssuedTotal.Inc()
}

// RecordRewardRedeemed records a reward being redeemed
func RecordRewardRedeemed() {
	Get().RewardsRedeemedTotal.Inc()
}

// RecordBudgetUtilization records budget utilization for a tenant/budget
func RecordBudgetUtilization(tenantID, budgetID string, utilized int64) {
	Get().BudgetUtilization.WithLabels(tenantID, budgetID).Set(utilized)
}

// RecordExternalAPICall records an external API call duration
func RecordExternalAPICall(duration time.Duration) {
	Get().ExternalAPILatency.Observe(duration)
}

// RecordCircuitBreakerState records circuit breaker state (0=closed, 1=open, 2=half-open)
func RecordCircuitBreakerState(service string, state int64) {
	Get().CircuitBreakerState.WithLabels(service).Set(state)
}

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(duration time.Duration, isError bool) {
	m := Get()
	m.HTTPRequestsTotal.Inc()
	m.HTTPRequestDuration.Observe(duration)
	if isError {
		m.HTTPErrorsTotal.Inc()
	}
}

// RecordActiveHTTPRequest increments active HTTP requests
func RecordActiveHTTPRequest() {
	Get().HTTPActiveRequests.Inc()
}

// RecordHTTPRequestComplete decrements active HTTP requests
func RecordHTTPRequestComplete() {
	Get().HTTPActiveRequests.Dec()
}
