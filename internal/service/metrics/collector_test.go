package metrics

import "testing"

func TestCollectorProxyMetrics(t *testing.T) {
	c := NewCollector()
	c.RecordRequest(10, false)
	c.RecordRequest(20, true)
	c.IncConnections()

	m := c.ProxyMetrics()
	if m.TotalRequests != 2 {
		t.Fatalf("expected 2 requests, got %d", m.TotalRequests)
	}
	if m.ErrorRate <= 0 {
		t.Fatal("expected positive error rate")
	}
	if m.ActiveConnections != 1 {
		t.Fatalf("expected 1 active connection, got %d", m.ActiveConnections)
	}
}

func TestCalcLatency(t *testing.T) {
	avg, p99 := calcLatency([]float64{1, 2, 3, 4, 100})
	if avg <= 0 || p99 <= 0 {
		t.Fatal("expected non-zero latency stats")
	}
}
