package dto

import (
	"fmt"
	"time"
)

type CounterfactualSubmitRequest struct {
	PatientID    *string                 `json:"patient_id,omitempty"`
	ModelVersion string                  `json:"model_version,omitempty"`
	Target       *CounterfactualTarget   `json:"target,omitempty"`
	Instance     *CounterfactualInstance `json:"instance"`
	Constraints  *CounterfactualRules    `json:"constraints"`
	Generation   *CounterfactualGenerate `json:"generation,omitempty"`
	Preferences  *CounterfactualPrefs    `json:"preferences,omitempty"`
}

type CounterfactualTarget struct {
	TargetClass          string  `json:"target_class,omitempty"`
	MinTargetProbability float64 `json:"min_target_probability,omitempty"`
}

type CounterfactualInstance struct {
	Features map[string]interface{} `json:"features"`
}

type CounterfactualRules struct {
	ImmutableFeatures     []string `json:"immutable_features,omitempty"`
	MutableAllowed        []string `json:"mutable_allowed,omitempty"`
	MustNotChange         []string `json:"must_not_change,omitempty"`
	MedicalRuleSetVersion string   `json:"medical_rule_set_version,omitempty"`
}

type CounterfactualGenerate struct {
	TotalCFs   int    `json:"total_cfs,omitempty"`
	Method     string `json:"method,omitempty"`
	RandomSeed int    `json:"random_seed,omitempty"`
	TimeoutMS  int    `json:"timeout_ms,omitempty"`
}

type CounterfactualPrefs struct {
	CostWeights      map[string]float64 `json:"cost_weights,omitempty"`
	ObjectiveWeights map[string]float64 `json:"objective_weights,omitempty"`
}

func (r *CounterfactualSubmitRequest) Validate() error {
	if r.Instance == nil {
		return fmt.Errorf("instance is required")
	}
	if len(r.Instance.Features) == 0 {
		return fmt.Errorf("instance.features is required")
	}
	if r.Constraints == nil {
		return fmt.Errorf("constraints is required")
	}

	if r.Target != nil {
		if r.Target.MinTargetProbability < 0 || r.Target.MinTargetProbability > 1 {
			return fmt.Errorf("target.min_target_probability must be between 0 and 1")
		}
	}

	immutable := make(map[string]bool, len(r.Constraints.ImmutableFeatures))
	for _, feature := range r.Constraints.ImmutableFeatures {
		immutable[feature] = true
	}
	for _, feature := range r.Constraints.MutableAllowed {
		if immutable[feature] {
			return fmt.Errorf("immutable and mutable overlap: %s", feature)
		}
	}

	if r.Generation != nil {
		if r.Generation.TotalCFs != 0 && (r.Generation.TotalCFs < 1 || r.Generation.TotalCFs > 20) {
			return fmt.Errorf("generation.total_cfs must be between 1 and 20")
		}
		if r.Generation.TimeoutMS != 0 && (r.Generation.TimeoutMS < 100 || r.Generation.TimeoutMS > 60000) {
			return fmt.Errorf("generation.timeout_ms must be between 100 and 60000")
		}
	}

	return nil
}

func (r *CounterfactualSubmitRequest) ToWorkerPayload(jobID string) map[string]interface{} {
	payload := map[string]interface{}{
		"request_id":  jobID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"instance":    r.Instance,
		"constraints": r.Constraints,
	}

	if r.PatientID != nil {
		payload["patient_id"] = *r.PatientID
	}
	if r.ModelVersion != "" {
		payload["model_version"] = r.ModelVersion
	}
	if r.Target != nil {
		payload["target"] = r.Target
	}
	if r.Generation != nil {
		payload["generation"] = r.Generation
	}
	if r.Preferences != nil {
		payload["preferences"] = r.Preferences
	}

	return payload
}
