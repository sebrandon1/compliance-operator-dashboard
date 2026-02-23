package compliance

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// --- Tier 1: Pure function tests ---

func TestResolveGVR(t *testing.T) {
	tests := []struct {
		name             string
		kind             string
		apiVersion       string
		defaultNamespace string
		wantGVR          schema.GroupVersionResource
		wantNamespace    string
		wantErr          bool
	}{
		{
			name:             "MachineConfig",
			kind:             "MachineConfig",
			apiVersion:       "machineconfiguration.openshift.io/v1",
			defaultNamespace: "openshift-compliance",
			wantGVR: schema.GroupVersionResource{
				Group: "machineconfiguration.openshift.io", Version: "v1", Resource: "machineconfigs",
			},
			wantNamespace: "", // cluster-scoped
		},
		{
			name:             "KubeletConfig",
			kind:             "KubeletConfig",
			apiVersion:       "machineconfiguration.openshift.io/v1",
			defaultNamespace: "openshift-compliance",
			wantGVR: schema.GroupVersionResource{
				Group: "machineconfiguration.openshift.io", Version: "v1", Resource: "kubeletconfigs",
			},
			wantNamespace: "",
		},
		{
			name:             "APIServer",
			kind:             "APIServer",
			apiVersion:       "config.openshift.io/v1",
			defaultNamespace: "openshift-compliance",
			wantGVR: schema.GroupVersionResource{
				Group: "config.openshift.io", Version: "v1", Resource: "apiservers",
			},
			wantNamespace: "",
		},
		{
			name:             "IngressController",
			kind:             "IngressController",
			apiVersion:       "operator.openshift.io/v1",
			defaultNamespace: "openshift-compliance",
			wantGVR: schema.GroupVersionResource{
				Group: "operator.openshift.io", Version: "v1", Resource: "ingresscontrollers",
			},
			wantNamespace: "",
		},
		{
			name:             "OAuth",
			kind:             "OAuth",
			apiVersion:       "config.openshift.io/v1",
			defaultNamespace: "openshift-compliance",
			wantGVR: schema.GroupVersionResource{
				Group: "config.openshift.io", Version: "v1", Resource: "oauths",
			},
			wantNamespace: "",
		},
		{
			name:             "ConfigMap",
			kind:             "ConfigMap",
			apiVersion:       "v1",
			defaultNamespace: "openshift-compliance",
			wantGVR: schema.GroupVersionResource{
				Group: "", Version: "v1", Resource: "configmaps",
			},
			wantNamespace: "openshift-compliance",
		},
		{
			name:             "Secret",
			kind:             "Secret",
			apiVersion:       "v1",
			defaultNamespace: "my-ns",
			wantGVR: schema.GroupVersionResource{
				Group: "", Version: "v1", Resource: "secrets",
			},
			wantNamespace: "my-ns",
		},
		{
			name:             "unknown kind uses default pluralization",
			kind:             "CustomThing",
			apiVersion:       "example.com/v1beta1",
			defaultNamespace: "test-ns",
			wantGVR: schema.GroupVersionResource{
				Group: "example.com", Version: "v1beta1", Resource: "customthings",
			},
			wantNamespace: "test-ns",
		},
		{
			name:             "core API version without group",
			kind:             "ConfigMap",
			apiVersion:       "v1",
			defaultNamespace: "default",
			wantGVR: schema.GroupVersionResource{
				Group: "", Version: "v1", Resource: "configmaps",
			},
			wantNamespace: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gvr, ns, err := resolveGVR(tt.kind, tt.apiVersion, tt.defaultNamespace)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveGVR() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if gvr != tt.wantGVR {
				t.Errorf("GVR = %v, want %v", gvr, tt.wantGVR)
			}
			if ns != tt.wantNamespace {
				t.Errorf("namespace = %q, want %q", ns, tt.wantNamespace)
			}
		})
	}
}

func TestDetectRoleFromObject(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		expected string
	}{
		{
			name: "role from label",
			obj: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]any{}}
				u.SetLabels(map[string]string{
					"machineconfiguration.openshift.io/role": "master",
				})
				return u
			}(),
			expected: "master",
		},
		{
			name: "master from name",
			obj: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]any{}}
				u.SetName("75-master-audit-rules")
				return u
			}(),
			expected: "master",
		},
		{
			name: "worker from name",
			obj: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]any{}}
				u.SetName("75-worker-sshd")
				return u
			}(),
			expected: "master", // "worker" contains "master"? No — let me re-check
			// detectRoleFromObject checks Contains(name, "master") first.
			// "75-worker-sshd" does not contain "master", so it falls through to return "worker".
		},
		{
			name: "fallback to worker",
			obj: func() *unstructured.Unstructured {
				u := &unstructured.Unstructured{Object: map[string]any{}}
				u.SetName("some-config")
				return u
			}(),
			expected: "worker",
		},
	}

	// Fix test case 2 — "75-worker-sshd" does not contain "master"
	tests[2].expected = "worker"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectRoleFromObject(tt.obj)
			if got != tt.expected {
				t.Errorf("detectRoleFromObject() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// --- Tier 2: Fake K8s client tests ---

func TestApplyRemediation(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	t.Run("applies ConfigMap remediation", func(t *testing.T) {
		rem := newRemediation("rem-cm", ns, map[string]any{
			"spec": map[string]any{
				"apply": false,
				"current": map[string]any{
					"object": map[string]any{
						"apiVersion": "v1",
						"kind":       "ConfigMap",
						"metadata": map[string]any{
							"name":      "test-configmap",
							"namespace": ns,
						},
						"data": map[string]any{
							"key1": "value1",
						},
					},
				},
			},
		})

		client := newTestClient(rem)

		result, err := ApplyRemediation(ctx, client, ns, "rem-cm")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Applied {
			t.Error("expected Applied=true")
		}
		if result.Message == "" {
			t.Error("expected non-empty message")
		}

		// Verify the ConfigMap was created in the fake client
		cmGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
		cm, err := client.Dynamic.Resource(cmGVR).Namespace(ns).
			Get(ctx, "test-configmap", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("ConfigMap not created: %v", err)
		}
		if cm.GetName() != "test-configmap" {
			t.Errorf("ConfigMap name = %q, want test-configmap", cm.GetName())
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := ApplyRemediation(ctx, nil, ns, "rem-1")
		if err == nil {
			t.Error("expected error for nil client")
		}
	})

	t.Run("missing remediation returns error", func(t *testing.T) {
		client := newTestClient()
		_, err := ApplyRemediation(ctx, client, ns, "nonexistent")
		if err == nil {
			t.Error("expected error for missing remediation")
		}
	})

	t.Run("remediation without object returns error", func(t *testing.T) {
		rem := newRemediation("rem-empty", ns, map[string]any{
			"spec": map[string]any{
				"apply": false,
			},
		})
		client := newTestClient(rem)

		_, err := ApplyRemediation(ctx, client, ns, "rem-empty")
		if err == nil {
			t.Error("expected error for remediation without spec.current.object")
		}
	})

	t.Run("remediation with missing kind returns error", func(t *testing.T) {
		rem := newRemediation("rem-no-kind", ns, map[string]any{
			"spec": map[string]any{
				"apply": false,
				"current": map[string]any{
					"object": map[string]any{
						"metadata": map[string]any{
							"name": "test",
						},
					},
				},
			},
		})
		client := newTestClient(rem)

		_, err := ApplyRemediation(ctx, client, ns, "rem-no-kind")
		if err == nil {
			t.Error("expected error for object missing kind/apiVersion")
		}
	})
}

func TestRemoveRemediation(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	t.Run("removes existing ConfigMap", func(t *testing.T) {
		// Create the remediation describing the target object
		rem := newRemediation("rem-cm", ns, map[string]any{
			"spec": map[string]any{
				"apply": true,
				"current": map[string]any{
					"object": map[string]any{
						"apiVersion": "v1",
						"kind":       "ConfigMap",
						"metadata": map[string]any{
							"name":      "target-cm",
							"namespace": ns,
						},
					},
				},
			},
		})

		// Pre-populate the target ConfigMap
		targetCM := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata": map[string]any{
					"name":      "target-cm",
					"namespace": ns,
				},
			},
		}

		client := newTestClient(rem, targetCM)

		result, err := RemoveRemediation(ctx, client, ns, "rem-cm")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Applied {
			t.Error("expected Applied=false after removal")
		}
		if result.Message == "" {
			t.Error("expected non-empty message")
		}

		// Verify the ConfigMap was deleted
		cmGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
		_, err = client.Dynamic.Resource(cmGVR).Namespace(ns).
			Get(ctx, "target-cm", metav1.GetOptions{})
		if err == nil {
			t.Error("expected ConfigMap to be deleted")
		}
	})

	t.Run("already removed object", func(t *testing.T) {
		rem := newRemediation("rem-gone", ns, map[string]any{
			"spec": map[string]any{
				"apply": false,
				"current": map[string]any{
					"object": map[string]any{
						"apiVersion": "v1",
						"kind":       "ConfigMap",
						"metadata": map[string]any{
							"name":      "already-gone",
							"namespace": ns,
						},
					},
				},
			},
		})

		client := newTestClient(rem)

		result, err := RemoveRemediation(ctx, client, ns, "rem-gone")
		// The fake client returns "not found" for missing objects, which
		// RemoveRemediation handles gracefully.
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Applied {
			t.Error("expected Applied=false")
		}
	})

	t.Run("nil client returns error", func(t *testing.T) {
		_, err := RemoveRemediation(ctx, nil, ns, "rem-1")
		if err == nil {
			t.Error("expected error for nil client")
		}
	})

	t.Run("missing remediation returns error", func(t *testing.T) {
		client := newTestClient()
		_, err := RemoveRemediation(ctx, client, ns, "nonexistent")
		if err == nil {
			t.Error("expected error for missing remediation")
		}
	})
}

func TestApplyRemediation_SetsApplyFlag(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	rem := newRemediation("rem-flag", ns, map[string]any{
		"spec": map[string]any{
			"apply": false,
			"current": map[string]any{
				"object": map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name":      "flagged-cm",
						"namespace": ns,
					},
				},
			},
		},
	})

	client := newTestClient(rem)

	_, err := ApplyRemediation(ctx, client, ns, "rem-flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that the remediation was updated with spec.apply=true
	updated, err := client.Dynamic.Resource(complianceRemediationGVR).Namespace(ns).
		Get(ctx, "rem-flag", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get updated remediation: %v", err)
	}

	applyVal, found, _ := unstructured.NestedBool(updated.Object, "spec", "apply")
	if !found {
		t.Error("spec.apply not found after ApplyRemediation")
	} else if !applyVal {
		t.Error("expected spec.apply=true after ApplyRemediation")
	}
}

func TestApplyRemediation_MachineConfig(t *testing.T) {
	ctx := context.Background()
	ns := "openshift-compliance"

	rem := newRemediation("rem-mc", ns, map[string]any{
		"spec": map[string]any{
			"apply": false,
			"current": map[string]any{
				"object": map[string]any{
					"apiVersion": "machineconfiguration.openshift.io/v1",
					"kind":       "MachineConfig",
					"metadata": map[string]any{
						"name": "75-worker-audit",
						"labels": map[string]any{
							"machineconfiguration.openshift.io/role": "worker",
						},
					},
				},
			},
		},
	})

	client := newTestClient(rem)

	result, err := ApplyRemediation(ctx, client, ns, "rem-mc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Applied {
		t.Error("expected Applied=true")
	}
	// MachineConfig message should include reboot hint
	if result.Message == "" {
		t.Error("expected non-empty message")
	}

	// Verify the MachineConfig was created (cluster-scoped)
	mcGVR := schema.GroupVersionResource{
		Group: "machineconfiguration.openshift.io", Version: "v1", Resource: "machineconfigs",
	}
	mc, err := client.Dynamic.Resource(mcGVR).Get(ctx, "75-worker-audit", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("MachineConfig not created: %v", err)
	}
	if mc.GetName() != "75-worker-audit" {
		t.Errorf("MachineConfig name = %q, want 75-worker-audit", mc.GetName())
	}
}
