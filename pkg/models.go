package pkg

type NodeResult struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	SubTitle string `json:"subtitle"`
	MainStat float64 `json:"mainstat"` // RPS
	SecondaryStat float64 `json:"secondarystat"` // Latency (p95)
	ArcSuccess float64 `json:"arc__success"`
	ArcError float64 `json:"arc__error"`
	DetailLatencyP50 float64 `json:"detail__latency_p50"`
	DetailLatencyAvg float64 `json:"detail__latency_avg"`
	DetailSuccessCount int64 `json:"detail__success_count"`
	DetailErrorCount int64 `json:"detail__error_count"`
	Icon string `json:"icon"`
	Highlighted bool `json:"highlighted"`
	NodeRadius int `json:"nodeRadius"`

	// used internally to build the graph
	TotalRequests int64 `json:"total_requests"`
}

type EdgeResult struct {
	ID string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	MainStat float64 `json:"mainstat"` // RPS
	SecondaryStat float64 `json:"secondarystat"` // Latency (p95)
	DetailLatencyP50 float64 `json:"detail__latency_p50"`
	DetailLatencyAvg float64 `json:"detail__latency_avg"`
	DetailSuccessCount int64 `json:"detail__success_count"`
	DetailErrorCount int64 `json:"detail__error_count"`
	Color string `json:"color"`
	Thickness int `json:"thickness"`
	StrokeDasharray string `json:"strokeDasharray"`

	// used internally to build the graph
	TotalRequests int64 `json:"total_requests"`
}
