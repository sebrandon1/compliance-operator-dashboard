package ws

import (
	"context"
	"log"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/sebrandon1/compliance-operator-dashboard/internal/k8s"
)

var watchedResources = []struct {
	GVR          schema.GroupVersionResource
	ResourceType string
}{
	{
		GVR:          schema.GroupVersionResource{Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "compliancecheckresults"},
		ResourceType: "ComplianceCheckResult",
	},
	{
		GVR:          schema.GroupVersionResource{Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "complianceremediations"},
		ResourceType: "ComplianceRemediation",
	},
	{
		GVR:          schema.GroupVersionResource{Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "compliancesuites"},
		ResourceType: "ComplianceSuite",
	},
	{
		GVR:          schema.GroupVersionResource{Group: "compliance.openshift.io", Version: "v1alpha1", Resource: "compliancescans"},
		ResourceType: "ComplianceScan",
	},
}

// Watcher bridges Kubernetes watch events to WebSocket broadcasts.
type Watcher struct {
	client    *k8s.Client
	hub       *Hub
	namespace string
}

// NewWatcher creates a new K8s Watch → WebSocket bridge.
func NewWatcher(client *k8s.Client, hub *Hub, namespace string) *Watcher {
	return &Watcher{
		client:    client,
		hub:       hub,
		namespace: namespace,
	}
}

// Start begins watching all compliance-related resources.
func (w *Watcher) Start(ctx context.Context) {
	for _, res := range watchedResources {
		go w.watchResource(ctx, res.GVR, res.ResourceType)
	}
}

func (w *Watcher) watchResource(ctx context.Context, gvr schema.GroupVersionResource, resourceType string) {
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		watcher, err := w.client.Dynamic.Resource(gvr).Namespace(w.namespace).
			Watch(ctx, metav1.ListOptions{})
		if err != nil {
			// If the CRD doesn't exist, back off much longer (operator not installed)
			if strings.Contains(err.Error(), "the server could not find the requested resource") ||
				strings.Contains(err.Error(), "no matches for kind") {
				log.Printf("CRD not found for %s, operator likely not installed (retrying in 60s)", resourceType)
				backoff = 60 * time.Second
			} else {
				log.Printf("Watch error for %s: %v (retrying in %v)", resourceType, err, backoff)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			// Exponential backoff, max 60 seconds
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			continue
		}

		// Reset backoff on successful watch
		backoff = time.Second
		w.processEvents(ctx, watcher, resourceType)
		watcher.Stop()
	}
}

func (w *Watcher) processEvents(ctx context.Context, watcher watch.Interface, resourceType string) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			obj, ok := event.Object.(*unstructured.Unstructured)
			if !ok {
				continue
			}

			var eventType WatchEventType
			switch event.Type {
			case watch.Added:
				eventType = WatchEventAdded
			case watch.Modified:
				eventType = WatchEventModified
			case watch.Deleted:
				eventType = WatchEventDeleted
			default:
				continue
			}

			// Determine the appropriate message type
			msgType := mapResourceToMessageType(resourceType, eventType)

			watchEvent := WatchEvent{
				EventType:    eventType,
				ResourceType: resourceType,
				Name:         obj.GetName(),
				Namespace:    obj.GetNamespace(),
				Data:         extractRelevantData(resourceType, obj),
			}

			w.hub.Broadcast(Message{
				Type:    msgType,
				Payload: watchEvent,
			})
		}
	}
}

func mapResourceToMessageType(resourceType string, _ WatchEventType) MessageType {
	switch resourceType {
	case "ComplianceCheckResult":
		return MessageTypeCheckResult
	case "ComplianceRemediation":
		return MessageTypeRemediation
	case "ComplianceSuite", "ComplianceScan":
		return MessageTypeScanStatus
	default:
		return MessageTypeError
	}
}

func extractRelevantData(resourceType string, obj *unstructured.Unstructured) map[string]interface{} {
	data := make(map[string]interface{})

	switch resourceType {
	case "ComplianceCheckResult":
		data["status"], _, _ = unstructured.NestedString(obj.Object, "status")
		data["severity"], _, _ = unstructured.NestedString(obj.Object, "severity")
		data["description"], _, _ = unstructured.NestedString(obj.Object, "description")

	case "ComplianceRemediation":
		kind, _, _ := unstructured.NestedString(obj.Object, "spec", "current", "object", "kind")
		apply, _, _ := unstructured.NestedString(obj.Object, "spec", "apply")
		data["kind"] = kind
		data["applied"] = apply == "true"

	case "ComplianceSuite":
		data["phase"], _, _ = unstructured.NestedString(obj.Object, "status", "phase")
		data["result"], _, _ = unstructured.NestedString(obj.Object, "status", "result")

	case "ComplianceScan":
		data["phase"], _, _ = unstructured.NestedString(obj.Object, "status", "phase")
		data["result"], _, _ = unstructured.NestedString(obj.Object, "status", "result")
	}

	return data
}
