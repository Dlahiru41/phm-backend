package growth

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	MetricWeightForAge = "weight_for_age"
	MetricHeightForAge = "height_for_age"
)

type ReferencePoint struct {
	AgeMonth int     `json:"ageMonth"`
	SDNeg3   float64 `json:"sdNeg3"`
	SDNeg2   float64 `json:"sdNeg2"`
	SDNeg1   float64 `json:"sdNeg1"`
	Median   float64 `json:"median"`
	SDPos1   float64 `json:"sdPos1"`
	SDPos2   float64 `json:"sdPos2"`
	SDPos3   float64 `json:"sdPos3"`
}

type fileFormat struct {
	Version    string                                 `json:"version"`
	Indicators map[string]map[string][]ReferencePoint `json:"indicators"`
	Metadata   map[string]string                      `json:"metadata,omitempty"`
}

type Assessor struct {
	version    string
	indicators map[string]map[string][]ReferencePoint
	metadata   map[string]string
}

func LoadAssessorFromFile(path string) (*Assessor, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read WHO reference: %w", err)
	}
	var parsed fileFormat
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("parse WHO reference: %w", err)
	}
	if len(parsed.Indicators) == 0 {
		return nil, fmt.Errorf("WHO reference contains no indicators")
	}
	for metric, bySex := range parsed.Indicators {
		for sex, points := range bySex {
			normalized := normalizeSex(sex)
			if normalized == "" {
				return nil, fmt.Errorf("unsupported sex key %q in metric %q", sex, metric)
			}
			sort.Slice(points, func(i, j int) bool { return points[i].AgeMonth < points[j].AgeMonth })
			parsed.Indicators[metric][normalized] = points
			if normalized != sex {
				delete(parsed.Indicators[metric], sex)
			}
		}
	}
	meta := map[string]string{}
	for k, v := range parsed.Metadata {
		meta[k] = v
	}
	return &Assessor{version: parsed.Version, indicators: parsed.Indicators, metadata: meta}, nil
}

func (a *Assessor) Version() string {
	if a == nil {
		return ""
	}
	return a.version
}

func (a *Assessor) HasStandardData() bool {
	if a == nil {
		return false
	}
	return len(a.indicators) > 0
}

func (a *Assessor) Series(metric, sex string) []ReferencePoint {
	if a == nil {
		return nil
	}
	bySex, ok := a.indicators[metric]
	if !ok {
		return nil
	}
	points := bySex[normalizeSex(sex)]
	out := make([]ReferencePoint, len(points))
	copy(out, points)
	return out
}

func (a *Assessor) Assess(metric, sex string, ageMonths int, value *float64) (string, *float64, bool) {
	if value == nil || a == nil {
		return "", nil, false
	}
	ref, ok := a.match(metric, sex, ageMonths)
	if !ok {
		return "", nil, false
	}
	z := zScoreFromReference(*value, ref)
	status := classify(metric, z)
	return status, &z, true
}

func (a *Assessor) match(metric, sex string, ageMonths int) (ReferencePoint, bool) {
	bySex, ok := a.indicators[metric]
	if !ok {
		return ReferencePoint{}, false
	}
	points := bySex[normalizeSex(sex)]
	if len(points) == 0 {
		return ReferencePoint{}, false
	}
	selected := points[0]
	for _, p := range points {
		if p.AgeMonth <= ageMonths {
			selected = p
			continue
		}
		break
	}
	return selected, true
}

func classify(metric string, z float64) string {
	switch metric {
	case MetricWeightForAge:
		if z < -2 {
			return "underweight"
		}
		if z > 2 {
			return "overweight"
		}
		return "normal"
	case MetricHeightForAge:
		if z < -2 {
			return "stunted"
		}
		return "normal"
	default:
		return ""
	}
}

func zScoreFromReference(v float64, r ReferencePoint) float64 {
	type knot struct {
		z float64
		v float64
	}
	knots := []knot{
		{z: -3, v: r.SDNeg3},
		{z: -2, v: r.SDNeg2},
		{z: -1, v: r.SDNeg1},
		{z: 0, v: r.Median},
		{z: 1, v: r.SDPos1},
		{z: 2, v: r.SDPos2},
		{z: 3, v: r.SDPos3},
	}
	if v <= knots[0].v {
		return linearZ(v, knots[0].z, knots[0].v, knots[1].z, knots[1].v)
	}
	for i := 0; i < len(knots)-1; i++ {
		if v <= knots[i+1].v {
			return linearZ(v, knots[i].z, knots[i].v, knots[i+1].z, knots[i+1].v)
		}
	}
	last := len(knots) - 1
	return linearZ(v, knots[last-1].z, knots[last-1].v, knots[last].z, knots[last].v)
}

func linearZ(value, z1, v1, z2, v2 float64) float64 {
	dv := v2 - v1
	if dv == 0 {
		return z1
	}
	ratio := (value - v1) / dv
	return z1 + ratio*(z2-z1)
}

func normalizeSex(sex string) string {
	s := strings.ToLower(strings.TrimSpace(sex))
	switch s {
	case "male", "m":
		return "male"
	case "female", "f":
		return "female"
	default:
		return ""
	}
}

func (a *Assessor) Metadata() map[string]string {
	if a == nil {
		return nil
	}
	copyMeta := map[string]string{}
	for k, v := range a.metadata {
		copyMeta[k] = v
	}
	return copyMeta
}
