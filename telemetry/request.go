package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/newrelic/newrelic-lambda-extension/lambda/extension/api"
	"github.com/newrelic/newrelic-lambda-extension/util"
)

// RequestContext is the Vortex request context
type RequestContext struct {
	FunctionName       string `json:"function_name"`
	InvokedFunctionARN string `json:"invoked_function_arn"`
	// Below are not relevant to Lambda Extensions, but ingest requires these to be present
	LogGroupName  string `json:"log_group_name"`
	LogStreamName string `json:"log_stream_name"`
}

// RequestData is the body of the Vortex request
type RequestData struct {
	Context RequestContext `json:"context"`
	Entry   string         `json:"entry"`
}

// LogsEntry is a CloudWatch Logs entry
type LogsEntry struct {
	LogEvents []LogsEvent `json:"logEvents"`
	// Below are not relevant to Lambda Extensions, but ingest expects these to be present
	LogGroup    string `json:"logGroup"`
	LogStream   string `json:"logStream"`
	MessageType string `json:"messageType"`
	Owner       string `json:"owner"`
}

// LogsEvent is a CloudWatch Logs event
type LogsEvent struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// BuildRequest builds a Vortex HTTP request
func BuildRequest(
	payload []byte,
	invocationEvent *api.InvocationEvent,
	functionName string,
	licenseKey string,
	url string,
	userAgent string,
) (*http.Request, error) {
	logEvent := LogsEvent{ID: util.UUID(), Message: string(payload), Timestamp: util.Timestamp()}
	logGroupName := fmt.Sprintf("/aws/lambda/%s", functionName)
	logEntry := LogsEntry{
		LogEvents: []LogsEvent{logEvent},
		LogGroup:  logGroupName,
	}

	entry, err := json.Marshal(logEntry)
	if err != nil {
		return nil, err
	}

	context := RequestContext{
		FunctionName:       functionName,
		InvokedFunctionARN: invocationEvent.InvokedFunctionARN,
		LogGroupName:       logGroupName,
		LogStreamName:      "placeholder",
	}
	data := RequestData{Context: context, Entry: string(entry)}

	uncompressed, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	compressed, err := util.Compress(uncompressed)
	if err != nil {
		return nil, fmt.Errorf("error compressing data: %v", err)
	}

	req, err := http.NewRequest("POST", url, compressed)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("X-License-Key", licenseKey)

	return req, nil
}
