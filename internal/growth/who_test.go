package growth

import "testing"

func TestAssessWeightAndHeight(t *testing.T) {
	assessor := &Assessor{indicators: map[string]map[string][]ReferencePoint{
		MetricWeightForAge: {
			"male": {
				{AgeMonth: 0, SDNeg3: 2, SDNeg2: 2.5, SDNeg1: 3, Median: 3.5, SDPos1: 4, SDPos2: 4.5, SDPos3: 5},
			},
		},
		MetricHeightForAge: {
			"male": {
				{AgeMonth: 0, SDNeg3: 44, SDNeg2: 46, SDNeg1: 48, Median: 50, SDPos1: 52, SDPos2: 54, SDPos3: 56},
			},
		},
	}}

	w := 4.6
	status, z, ok := assessor.Assess(MetricWeightForAge, "male", 0, &w)
	if !ok || z == nil {
		t.Fatalf("expected successful weight assessment")
	}
	if status != "overweight" {
		t.Fatalf("unexpected weight status: %s", status)
	}

	h := 45.0
	status, z, ok = assessor.Assess(MetricHeightForAge, "male", 0, &h)
	if !ok || z == nil {
		t.Fatalf("expected successful height assessment")
	}
	if status != "stunted" {
		t.Fatalf("unexpected height status: %s", status)
	}
}
