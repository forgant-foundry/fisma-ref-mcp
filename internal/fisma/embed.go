package fisma

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/fy2025-ig-fisma-metrics.json
var metricsJSON []byte

// ContextMarkdown is the document-level guidance from the FY 2025 IG FISMA
// Metrics Evaluator's Guide: maturity model explanation, core/supplemental
// metric definitions, terms, alternative evidence considerations, and
// recommendations guidance. It covers content that applies across all metrics
// rather than to any individual one.
//
//go:embed data/fy2025-ig-fisma-metrics-context.md
var ContextMarkdown []byte

// Load parses and returns all 35 FY 2025 IG FISMA metrics from the embedded JSON.
func Load() ([]Metric, error) {
	var metrics []Metric
	if err := json.Unmarshal(metricsJSON, &metrics); err != nil {
		return nil, fmt.Errorf("parse fisma metrics: %w", err)
	}
	return metrics, nil
}
