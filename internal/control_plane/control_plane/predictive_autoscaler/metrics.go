package predictive_autoscaler

// Stat defines a single measurement at a point in time.
type Stat struct {
	pod_name                            string
	average_concurrent_requests         float64
	average_proxied_concurrent_requests float64
	request_count                       float64
	proxied_request_count               float64
	process_uptime                      float64
	timestamp                           int64
}
