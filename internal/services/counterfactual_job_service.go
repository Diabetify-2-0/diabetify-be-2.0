package services

import (
	"diabetify/internal/models"
	"diabetify/internal/repository"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type CounterfactualJobService interface {
	Start() error
	Stop()
	SubmitJob(jobID string, payload map[string]interface{}) error
	GetStatus() map[string]interface{}
}

type counterfactualJobService struct {
	repo repository.CounterfactualJobRepository

	rabbitURL     string
	requestQueue  string
	responseQueue string

	conn        *amqp.Connection
	pubChan     *amqp.Channel
	consChan    *amqp.Channel
	consumerTag string

	running bool
	mu      sync.RWMutex
	wg      sync.WaitGroup
}

func NewCounterfactualJobService(
	repo repository.CounterfactualJobRepository,
	rabbitURL string,
	requestQueue string,
	responseQueue string,
) CounterfactualJobService {
	if requestQueue == "" {
		requestQueue = "ml.cf.request"
	}
	if responseQueue == "" {
		responseQueue = "ml.cf.response"
	}

	return &counterfactualJobService{
		repo:          repo,
		rabbitURL:     rabbitURL,
		requestQueue:  requestQueue,
		responseQueue: responseQueue,
		consumerTag:   "cf_response_handler",
	}
}

func (s *counterfactualJobService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}

	conn, err := amqp.Dial(s.rabbitURL)
	if err != nil {
		return fmt.Errorf("failed to connect RabbitMQ: %w", err)
	}

	pubChan, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open publish channel: %w", err)
	}

	consChan, err := conn.Channel()
	if err != nil {
		pubChan.Close()
		conn.Close()
		return fmt.Errorf("failed to open consume channel: %w", err)
	}

	if _, err := pubChan.QueueDeclare(s.requestQueue, true, false, false, false, nil); err != nil {
		consChan.Close()
		pubChan.Close()
		conn.Close()
		return fmt.Errorf("failed to declare request queue: %w", err)
	}
	if _, err := consChan.QueueDeclare(s.responseQueue, true, false, false, false, nil); err != nil {
		consChan.Close()
		pubChan.Close()
		conn.Close()
		return fmt.Errorf("failed to declare response queue: %w", err)
	}
	if err := consChan.Qos(1, 0, false); err != nil {
		consChan.Close()
		pubChan.Close()
		conn.Close()
		return fmt.Errorf("failed to set qos: %w", err)
	}

	msgs, err := consChan.Consume(s.responseQueue, s.consumerTag, false, false, false, false, nil)
	if err != nil {
		consChan.Close()
		pubChan.Close()
		conn.Close()
		return fmt.Errorf("failed to consume response queue: %w", err)
	}

	s.conn = conn
	s.pubChan = pubChan
	s.consChan = consChan
	s.running = true

	s.wg.Add(1)
	go s.handleResponses(msgs)

	return nil
}

func (s *counterfactualJobService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	if s.consChan != nil {
		_ = s.consChan.Cancel(s.consumerTag, false)
		_ = s.consChan.Close()
	}
	if s.pubChan != nil {
		_ = s.pubChan.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.wg.Wait()
}

func (s *counterfactualJobService) SubmitJob(jobID string, payload map[string]interface{}) error {
	s.mu.RLock()
	running := s.running
	pubChan := s.pubChan
	s.mu.RUnlock()

	if !running || pubChan == nil {
		return fmt.Errorf("counterfactual service is not running")
	}

	payload["request_id"] = jobID
	if _, ok := payload["timestamp"]; !ok {
		payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = pubChan.Publish(
		"",
		s.requestQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          body,
			CorrelationId: jobID,
			ReplyTo:       s.responseQueue,
			DeliveryMode:  amqp.Persistent,
			Timestamp:     time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish counterfactual request: %w", err)
	}

	if err := s.repo.UpdateJobSubmissionStatus(jobID); err != nil {
		log.Printf("warning: request published but failed updating job submission status for %s: %v", jobID, err)
	}

	return nil
}

func (s *counterfactualJobService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"running":            s.running,
		"request_queue":      s.requestQueue,
		"response_queue":     s.responseQueue,
		"rabbitmq_connected": s.conn != nil && !s.conn.IsClosed(),
	}
}

func (s *counterfactualJobService) handleResponses(msgs <-chan amqp.Delivery) {
	defer s.wg.Done()
	for msg := range msgs {
		s.handleSingleResponse(msg)
	}
}

func (s *counterfactualJobService) handleSingleResponse(msg amqp.Delivery) {
	jobID := msg.CorrelationId
	if jobID == "" {
		var fallback map[string]interface{}
		if err := json.Unmarshal(msg.Body, &fallback); err == nil {
			if requestID, ok := fallback["request_id"].(string); ok {
				jobID = requestID
			}
		}
	}
	if jobID == "" {
		_ = msg.Ack(false)
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		errMsg := fmt.Sprintf("invalid counterfactual response payload: %v", err)
		_ = s.repo.UpdateJobTerminalStatus(jobID, models.CFJobStatusFailed, nil, &errMsg, nil)
		_ = msg.Ack(false)
		return
	}

	rawStatus, _ := payload["status"].(string)
	reasonCode, _ := payload["reason_code"].(string)
	message, _ := payload["message"].(string)
	resultStr := string(msg.Body)

	var finalStatus string
	switch rawStatus {
	case "FEASIBLE":
		finalStatus = models.CFJobStatusCompleted
	case "INFEASIBLE":
		finalStatus = models.CFJobStatusInfeasible
	default:
		finalStatus = models.CFJobStatusFailed
	}

	var reasonCodePtr *string
	if reasonCode != "" {
		reasonCodePtr = &reasonCode
	}
	var messagePtr *string
	if message != "" {
		messagePtr = &message
	}

	if err := s.repo.UpdateJobTerminalStatus(jobID, finalStatus, reasonCodePtr, messagePtr, &resultStr); err != nil {
		log.Printf("failed to update counterfactual job result for %s: %v", jobID, err)
	}

	_ = msg.Ack(false)
}
