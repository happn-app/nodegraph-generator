package pkg

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	metricReqTotal         = "traces_service_graph_request_total"
	metricReqFailedTotal   = "traces_service_graph_request_failed_total"
	metricReqLatencyBucket = "traces_service_graph_request_server_seconds_bucket"
	metricReqLatencySum    = "traces_service_graph_request_server_seconds_sum"
	metricReqLatencyCount  = "traces_service_graph_request_server_seconds_count"
)

type vmClient struct {
	ctx context.Context
	api v1.API
}

func New(host string, ctx context.Context) (*vmClient, error) {
	if (!strings.HasPrefix(host, "http://")) && (!strings.HasPrefix(host, "https://")) {
		host = "http://" + host
	}
	hostUri, err := url.Parse(host)
	log.Printf(
		"Creating new VMClient for host=%q uri.scheme=%q uri.host=%q uri.path=%q",
		host, hostUri.Scheme, hostUri.Host, hostUri.Path,
	)
	if err != nil {
		return nil, err
	}
	if hostUri.Scheme == "" {
		log.Printf("No scheme found in host URL, defaulting to http")
		hostUri, err = url.Parse("http://" + hostUri.String())
		if err != nil {
			return nil, err
		}
	}
	if hostUri.Path != "select/0/prometheus" {
		log.Printf("No Prometheus path found in host URL, adding /select/0/prometheus")
		hostUri = hostUri.ResolveReference(&url.URL{Path: "select/0/prometheus"})
	}

	client, err := api.NewClient(api.Config{
		Address: hostUri.String(),
	})
	if err != nil {
		return nil, err
	}
	log.Printf("Creating new API client to connect to %s", host)
	api := v1.NewAPI(client)
	return &vmClient{
		ctx: ctx,
		api: api,
	}, nil
}

func (client vmClient) QueryVectorLastInRange(expr string, rng time.Duration, step time.Duration) (model.Vector, error) {
	if step <= 0 {
		// some reasonable default granularity
		step = rng / 10
		if step <= 0 {
			step = time.Second
		}
	}

	end := time.Now()
	start := end.Add(-rng)

	log.Printf("[VMClient] QueryVectorLastInRange: expr=%q, start=%s, end=%s, step=%s",
		expr, start.Format(time.RFC3339), end.Format(time.RFC3339), step)

	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}

	val, _, err := client.api.QueryRange(client.ctx, expr, r)
	if err != nil {
		return nil, err
	}

	if val.Type() != model.ValMatrix {
		return nil, fmt.Errorf("expected matrix for %q, got %s", expr, val.Type().String())
	}

	matrix := val.(model.Matrix)
	vec := make(model.Vector, 0, len(matrix))

	for _, stream := range matrix {
		if len(stream.Values) == 0 {
			continue
		}

		// last sample in the range for this time series
		last := stream.Values[len(stream.Values)-1]

		sample := &model.Sample{
			Metric:    stream.Metric,
			Value:     last.Value,
			Timestamp: last.Timestamp,
		}

		vec = append(vec, sample)
	}

	return vec, nil
}

func (client vmClient) QueryVector(expr string, instant bool) (model.Vector, error) {
	log.Printf("[VMClient] QueryVector: expr=%q instant=%t", expr, instant)
	if !instant {
		return client.QueryVectorLastInRange(expr, 24*time.Hour, 1*time.Minute)
	}
	val, _, err := client.api.Query(client.ctx, expr, time.Now())
	if err != nil {
		return nil, err
	}
	if val.Type() != model.ValVector {
		return nil, fmt.Errorf("expected vector for %q, got %s", expr, val.Type().String())
	}
	return val.(model.Vector), nil
}

func (client vmClient) GetFullGraph(env string) (map[string]*NodeResult, map[string]*EdgeResult, error) {
	log.Printf("[VMClient] GetFullGraph: env=%q", env)

	nodes := make(map[string]*NodeResult)
	edges := make(map[string]*EdgeResult)

	vector, err := client.QueryVector(fmt.Sprintf("%s{env=%q}", metricReqTotal, env), false)
	if err != nil {
		return nodes, edges, err
	}

	for _, sample := range vector {
		client := MapServiceName(string(sample.Metric["client"]))
		server := MapServiceName(string(sample.Metric["server"]))
		connectionType := string(sample.Metric["connection_type"])
		if client == "" || server == "" {
			continue
		}

		edgeKey := EdgeKey(sample)
		edge, exists := edges[edgeKey]
		if !exists {
			edge = &EdgeResult{
				ID:               edgeKey,
				Source:           client,
				Target:           server,
				DetailErrorCount: 0,
				StrokeDasharray:  "0", // solid line by default
				Thickness:        1,
				Color:            "#999",
			}
			edges[edgeKey] = edge
		}
		edge.TotalRequests = int64(sample.Value)
		if _, exists := nodes[server]; !exists {
			nodes[server] = &NodeResult{
				ID:            server,
				Title:         server,
				SubTitle:      env,
				Icon:          ConnectionTypeToIcon(connectionType, server),
				TotalRequests: int64(sample.Value),
				ArcSuccess:    1,
				ArcError:      0,
				NodeRadius:    30,
				Highlighted:   false,
			}
		} else {
			nodes[server].TotalRequests += int64(sample.Value)
		}
		if _, exists := nodes[client]; !exists {
			nodes[client] = &NodeResult{
				ID:               client,
				Title:            client,
				SubTitle:         env,
				Icon:             ConnectionTypeToIcon(connectionType, client),
				TotalRequests:    int64(sample.Value),
				DetailErrorCount: 0,
				NodeRadius:       30,
				ArcSuccess:       1,
				ArcError:         0,
				Highlighted:      false,
			}
		} else {
			nodes[client].TotalRequests += int64(sample.Value)
		}
	}
	log.Printf("[VMClient] GetFullGraph: collected %d nodes and %d edges", len(nodes), len(edges))
	return nodes, edges, nil
}

func (client vmClient) EnrichWithQuantiles(env string, edges map[string]*EdgeResult, nodes map[string]*NodeResult) error {
	log.Printf("[VMClient] EnrichWithQuantiles: querying p95 latency for all edges for env=%q", env)
	vector, err := client.QueryVector(
		fmt.Sprintf(
			"histogram_quantile(0.95, sum(rate(%s{env=%q}[5m])) by (le, server, client))",
			metricReqLatencyBucket,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		edgeKey := EdgeKey(sample)
		if edge, exists := edges[edgeKey]; exists {
			edge.SecondaryStat = float64(sample.Value)
		}
	}
	log.Printf("[VMClient] EnrichWithQuantiles: querying p95 latency for all nodes for env=%q", env)
	vector, err = client.QueryVector(
		fmt.Sprintf(
			"histogram_quantile(0.95, sum(rate(%s{env=%q}[5m])) by (le, server))",
			metricReqLatencyBucket,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		server := MapServiceName(string(sample.Metric["server"]))
		if node, exists := nodes[server]; exists {
			node.SecondaryStat = float64(sample.Value)
		}
	}

	log.Printf("[VMClient] EnrichWithQuantiles: querying p50 latency for all edges for env=%q", env)
	vector, err = client.QueryVector(
		fmt.Sprintf(
			"histogram_quantile(0.50, sum(rate(%s{env=%q}[5m])) by (le, server, client))",
			metricReqLatencyBucket,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		edgeKey := EdgeKey(sample)
		if edge, exists := edges[edgeKey]; exists {
			edge.DetailLatencyP50 = float64(sample.Value)
		}
	}
	log.Printf("[VMClient] EnrichWithQuantiles: querying p50 latency for all nodes for env=%q", env)
	vector, err = client.QueryVector(
		fmt.Sprintf(
			"histogram_quantile(0.50, sum(rate(%s{env=%q}[5m])) by (le, server))",
			metricReqLatencyBucket,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		server := MapServiceName(string(sample.Metric["server"]))
		if node, exists := nodes[server]; exists {
			node.DetailLatencyP50 = float64(sample.Value)
		}
	}
	return nil
}

func (client vmClient) EnrichWithAverages(env string, edges map[string]*EdgeResult, nodes map[string]*NodeResult) error {
	log.Printf("[VMClient] EnrichWithAverages: querying avg latency for all edges for env=%q", env)
	vector, err := client.QueryVector(
		fmt.Sprintf(
			"rate(%s{env=%q}[5m]) / rate(%s{env=%q}[5m])",
			metricReqLatencySum,
			env,
			metricReqLatencyCount,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		edgeKey := EdgeKey(sample)
		if edge, exists := edges[edgeKey]; exists {
			edge.DetailLatencyAvg = float64(sample.Value)
		}
	}
	log.Printf("[VMClient] EnrichWithAverages: querying avg latency for all nodes for env=%q", env)
	vector, err = client.QueryVector(
		fmt.Sprintf(
			"sum(rate(%s{env=%q}[5m]) / rate(%s{env=%q}[5m])) by (server)",
			metricReqLatencySum,
			env,
			metricReqLatencyCount,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		server := MapServiceName(string(sample.Metric["server"]))
		if node, exists := nodes[server]; exists {
			node.DetailLatencyAvg = float64(sample.Value)
		}
	}
	return nil
}

func (client vmClient) EnrichWithRatePerSecond(env string, edges map[string]*EdgeResult, nodes map[string]*NodeResult) error {
	log.Printf("[VMClient] EnrichWithRatePerSecond: querying RPS for all edges for env=%q", env)
	vector, err := client.QueryVector(
		fmt.Sprintf(
			"rate(%s{env=%q}[5m])",
			metricReqTotal,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		edgeKey := EdgeKey(sample)
		if edge, exists := edges[edgeKey]; exists {
			edge.MainStat = float64(sample.Value)
		}
	}
	log.Printf("[VMClient] EnrichWithRatePerSecond: querying RPS for all nodes for env=%q", env)
	vector, err = client.QueryVector(
		fmt.Sprintf(
			"sum(rate(%s{env=%q}[5m])) by (server)",
			metricReqTotal,
			env,
		),
		false,
	)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		server := MapServiceName(string(sample.Metric["server"]))
		if node, exists := nodes[server]; exists {
			node.MainStat = float64(sample.Value)
		}
	}
	return nil
}

func (client vmClient) EnrichGraph(nodes map[string]*NodeResult, edges map[string]*EdgeResult, env string) error {
	log.Printf("[VMClient] EnrichGraph: enriching %d edges and %d nodes for env=%q", len(edges), len(nodes), env)
	if len(edges) == 0 {
		// Implies no data, len(nodes) should be 0 as well
		return nil
	}
	log.Printf("[VMClient] EnrichGraph: querying failed requests for env=%q", env)
	vector, err := client.QueryVector(fmt.Sprintf("%s{env=%q}", metricReqFailedTotal, env), false)
	if err != nil {
		return err
	}
	for _, sample := range vector {
		target := MapServiceName(string(sample.Metric["server"]))
		edgeKey := EdgeKey(sample)
		if edge, exists := edges[edgeKey]; exists {
			edge.DetailErrorCount = int64(sample.Value)
			edge.DetailSuccessCount = edge.TotalRequests - edge.DetailErrorCount
		}
		if node, exists := nodes[target]; exists {
			node.DetailErrorCount += int64(sample.Value)
			node.DetailSuccessCount = node.TotalRequests - node.DetailErrorCount
			node.ArcSuccess = float64(node.DetailSuccessCount) / float64(node.TotalRequests)
			node.ArcError = float64(node.DetailErrorCount) / float64(node.TotalRequests)
		}
	}

	err = client.EnrichWithQuantiles(env, edges, nodes)
	if err != nil {
		return err
	}
	err = client.EnrichWithAverages(env, edges, nodes)
	if err != nil {
		return err
	}
	err = client.EnrichWithRatePerSecond(env, edges, nodes)
	if err != nil {
		return err
	}

	return nil
}
