package lightspeed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

// NewQuerier returns a sensor component that handles Lightspeed query requests.
func NewQuerier(updater *updaterImpl) common.SensorComponent {
	ctx, ctxCancel := context.WithCancel(context.Background())

	q := &querierImpl{
		responsesC:   make(chan *message.ExpiringMessage),
		requestQueue: make(chan *central.LightspeedQueryRequest, 10),
		updater:      updater,
		ctx:          ctx,
		ctxCancel:    ctxCancel,
	}

	return q
}

type querierImpl struct {
	responsesC   chan *message.ExpiringMessage
	requestQueue chan *central.LightspeedQueryRequest
	updater      *updaterImpl
	ctx          context.Context
	ctxCancel    context.CancelFunc
}

func (q *querierImpl) Name() string {
	return "lightspeed.querierImpl"
}

func (q *querierImpl) Start() error {
	go q.processRequests()
	return nil
}

func (q *querierImpl) Stop() {
	q.ctxCancel()
}

func (q *querierImpl) Notify(_ common.SensorComponentEvent) {}

func (q *querierImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (q *querierImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return q.responsesC
}

func (q *querierImpl) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetLightspeedQueryRequest() != nil
}

func (q *querierImpl) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	req := msg.GetLightspeedQueryRequest()
	if req == nil {
		return nil
	}

	select {
	case q.requestQueue <- req:
		log.Debugf("Enqueued Lightspeed query request: id=%s", req.GetId())
		return nil
	default:
		return fmt.Errorf("Lightspeed query queue is full")
	}
}

func (q *querierImpl) processRequests() {
	for {
		select {
		case <-q.ctx.Done():
			return
		case req, more := <-q.requestQueue:
			if !more {
				return
			}
			q.handleQuery(req)
		}
	}
}

func (q *querierImpl) handleQuery(req *central.LightspeedQueryRequest) {
	response := &central.LightspeedQueryResponse{
		Id: req.GetId(),
	}

	host := q.updater.GetHost()
	if host == "" {
		response.Error = "Lightspeed host not configured"
		q.sendResponse(response)
		return
	}

	token, err := q.updater.readSAToken()
	if err != nil {
		response.Error = fmt.Sprintf("failed to read SA token: %v", err)
		q.sendResponse(response)
		return
	}

	summary, err := q.queryLightspeed(host, token, req)
	if err != nil {
		response.Error = fmt.Sprintf("query failed: %v", err)
		q.sendResponse(response)
		return
	}

	response.Summary = summary
	q.sendResponse(response)
}

func (q *querierImpl) queryLightspeed(host, token string, req *central.LightspeedQueryRequest) (string, error) {
	url := fmt.Sprintf("%s/v1/query", host)

	// Combine query and context_json into the query field
	combinedQuery := req.GetQuery()
	if contextJSON := req.GetContextJson(); contextJSON != "" {
		combinedQuery = fmt.Sprintf("%s\n\nContext: %s", combinedQuery, contextJSON)
	}

	requestBody := map[string]string{
		"query": combinedQuery,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := q.updater.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var responseBody struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&responseBody); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return responseBody.Response, nil
}

func (q *querierImpl) sendResponse(response *central.LightspeedQueryResponse) {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_LightspeedQueryResponse{
			LightspeedQueryResponse: response,
		},
	}

	select {
	case q.responsesC <- message.New(msg):
		log.Debugf("Sent Lightspeed query response: id=%s, error=%s", response.GetId(), response.GetError())
	case <-q.ctx.Done():
		log.Warn("Failed to send Lightspeed query response: context cancelled")
	}
}
