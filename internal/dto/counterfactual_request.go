package dto

import (
	"fmt"
	"time"
)

type CounterfactualSubmitRequest struct {
	Target      *CounterfactualTarget   `json:"target,omitempty"`
	Instance    *CounterfactualInstance `json:"instance"`
	Constraints *CounterfactualRules    `json:"constraints"`
	Generation  *CounterfactualGenerate `json:"generation,omitempty"`
}

type CounterfactualTarget struct {
	TargetClass          string  `json:"target_class,omitempty"`
	MinTargetProbability float64 `json:"min_target_probability,omitempty"`
}

type CounterfactualInstance struct {
	Features map[string]interface{} `json:"features"`
}

type CounterfactualRules struct {
	MutableAllowed []string `json:"mutable_allowed,omitempty"`
}

type CounterfactualGenerate struct {
	TimeoutMS int `json:"timeout_ms,omitempty"`
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

	if r.Generation != nil {
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

	if r.Target != nil {
		payload["target"] = r.Target
	}
	if r.Generation != nil && r.Generation.TimeoutMS != 0 {
		payload["generation"] = r.Generation
	}

	return payload
}
