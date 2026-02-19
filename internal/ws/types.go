package ws

// MessageType identifies the kind of WebSocket message.
type MessageType string

const (
	MessageTypeClusterStatus   MessageType = "cluster_status"
	MessageTypeOperatorStatus  MessageType = "operator_status"
	MessageTypeInstallProgress MessageType = "install_progress"
	MessageTypeScanStatus      MessageType = "scan_status"
	MessageTypeCheckResult     MessageType = "check_result"
	MessageTypeRemediation     MessageType = "remediation"
	MessageTypeRemediationResult MessageType = "remediation_result"
	MessageTypeError           MessageType = "error"
)

// Message is a typed WebSocket message envelope.
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

// WatchEventType maps to K8s watch event types.
type WatchEventType string

const (
	WatchEventAdded    WatchEventType = "ADDED"
	WatchEventModified WatchEventType = "MODIFIED"
	WatchEventDeleted  WatchEventType = "DELETED"
)

// WatchEvent wraps a K8s watch event for WebSocket delivery.
type WatchEvent struct {
	EventType    WatchEventType `json:"event_type"`
	ResourceType string         `json:"resource_type"`
	Name         string         `json:"name"`
	Namespace    string         `json:"namespace"`
	Data         interface{}    `json:"data,omitempty"`
}
