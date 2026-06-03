package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type xaiFeaturePayload struct {
	Feature      string  `json:"feature"`
	Alias        string  `json:"alias"`
	Value        float64 `json:"value"`
	Shap         float64 `json:"shap"`
	Contribution float64 `json:"contribution"`
	Impact       string  `json:"impact"`
	Explanation  string  `json:"explanation"`
}

type xaiResultsPayload struct {
	UserID    string              `json:"user_id"`
	RiskScore float64             `json:"risk_score"`
	Features  []xaiFeaturePayload `json:"features"`
}

var featureAliases = map[string]string{
	"age":                                  "Usia",
	"BMI":                                  "Indeks Massa Tubuh (BMI)",
	"brinkman_index":                       "Indeks Brinkman",
	"brinkman_score":                       "Indeks Brinkman",
	"is_hypertension":                      "Riwayat Hipertensi",
	"is_cholesterol":                       "Riwayat Kolesterol Tinggi",
	"is_bloodline":                         "Riwayat Keluarga Diabetes",
	"is_macrosomic_baby":                   "Riwayat Melahirkan Bayi Makrosomia",
	"smoking_status":                       "Status Merokok",
	"moderate_physical_activity_frequency": "Frekuensi Aktivitas Fisik",
	"physical_activity_frequency":          "Frekuensi Aktivitas Fisik",
}

func chatbotServiceURL() string {
	if u := os.Getenv("CHATBOT_SERVICE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	if u := os.Getenv("CHATBOT_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://localhost:8023"
}

func mapXaiImpact(shap, contribution float64) string {
	magnitude := math.Abs(contribution)
	if shap >= 0 {
		switch {
		case magnitude >= 0.15:
			return "high_positive"
		case magnitude >= 0.05:
			return "medium_positive"
		default:
			return "low_positive"
		}
	}
	switch {
	case magnitude >= 0.15:
		return "high_negative"
	case magnitude >= 0.05:
		return "medium_negative"
	default:
		return "low_negative"
	}
}

func normalizeFeatureKey(key string) string {
	switch key {
	case "brinkman_index":
		return "brinkman_score"
	case "moderate_physical_activity_frequency":
		return "physical_activity_frequency"
	default:
		return key
	}
}

func buildXaiExplanation(alias string, value float64, impact string) string {
	direction := "meningkatkan"
	if impact == "high_negative" || impact == "medium_negative" || impact == "low_negative" {
		direction = "menurunkan"
	}
	return fmt.Sprintf(
		"%s Anda (nilai: %g) berkontribusi %s risiko diabetes berdasarkan analisis SHAP terbaru.",
		alias, value, direction,
	)
}

func buildXaiPayload(userID uint, rabbitResponse *RabbitMQPredictionResponse) xaiResultsPayload {
	type scoredFeature struct {
		payload xaiFeaturePayload
		weight  float64
	}

	scored := make([]scoredFeature, 0, len(rabbitResponse.Explanation))

	for featureName, featureData := range rabbitResponse.Explanation {
		shap, _ := featureData["shap"].(float64)
		contribution, _ := featureData["contribution"].(float64)
		if contribution == 0 {
			contribution = shap
		}

		var value float64
		switch v := featureData["value"].(type) {
		case float64:
			value = v
		case int:
			value = float64(v)
		}

		normalizedKey := normalizeFeatureKey(featureName)
		alias := featureAliases[featureName]
		if alias == "" {
			alias = featureAliases[normalizedKey]
		}
		if alias == "" {
			alias = normalizedKey
		}

		impact := mapXaiImpact(shap, contribution)
		scored = append(scored, scoredFeature{
			payload: xaiFeaturePayload{
				Feature:      normalizedKey,
				Alias:        alias,
				Value:        value,
				Shap:         shap,
				Contribution: contribution,
				Impact:       impact,
				Explanation:  buildXaiExplanation(alias, value, impact),
			},
			weight: math.Abs(contribution),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].weight > scored[j].weight
	})

	features := make([]xaiFeaturePayload, len(scored))
	for i, sf := range scored {
		features[i] = sf.payload
	}

	return xaiResultsPayload{
		UserID:    fmt.Sprintf("%d", userID),
		RiskScore: rabbitResponse.Prediction,
		Features:  features,
	}
}

// syncXaiToChatbot POSTs the latest SHAP profile to the chatbot service (stored in PostgreSQL).
func syncXaiToChatbot(userID uint, rabbitResponse *RabbitMQPredictionResponse) {
	if rabbitResponse == nil {
		return
	}

	payload := buildXaiPayload(userID, rabbitResponse)
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Warning: failed to marshal XAI payload for user %d: %v\n", userID, err)
		return
	}

	url := chatbotServiceURL() + "/api/v1/xai"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Warning: failed to create XAI sync request for user %d: %v\n", userID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("XAI_INTERNAL_API_KEY"); key != "" {
		req.Header.Set("X-Internal-Key", key)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Warning: failed to sync XAI to chatbot for user %d: %v\n", userID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf(
			"Warning: chatbot XAI sync for user %d returned HTTP %d\n",
			userID, resp.StatusCode,
		)
		return
	}

	fmt.Printf("Synced XAI profile for user %d to chatbot via HTTP\n", userID)
}
