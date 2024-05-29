package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodeCondition(t *testing.T) {
	testTable := map[string]struct {
		Node     *corev1.Node
		Expected string
	}{
		"Basic Case": {
			Node: &corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{
							Type:   corev1.NodeConditionType("Ready"),
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			Expected: "Ready",
		},
	}

	for name, tc := range testTable {
		t.Run(name, func(t *testing.T) {
			actual := nodeCondition(*tc.Node)
			assert.Equal(t, tc.Expected, actual, "failed")
		})
	}
}

func TestNodepoolLabels(t *testing.T) {
	testTable := map[string]struct {
		Node        *corev1.Node
		CustomLabel string
		Expected    string
	}{
		"no matching label": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"custom-label": "custom label",
					},
				},
			},
			Expected: "-",
		},
		"custom label": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"custom-label": "custom label",
					},
				},
			},
			CustomLabel: "custom-label",
			Expected:    "custom label",
		},
		"AWS": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"eks.amazonaws.com/nodegroup": "test AWS",
					},
				},
			},
			Expected: "test AWS",
		},
		"GCP": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"cloud.google.com/gke-nodepool": "test GCP",
					},
				},
			},
			Expected: "test GCP",
		},
		"AKS": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.azure.com/agentpool": "test AKS",
					},
				},
			},
			Expected: "test AKS",
		},
		"DigitalOcean": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"doks.digitalocean.com/node-pool-id": "test DigitalOcean",
					},
				},
			},
			Expected: "test DigitalOcean",
		},
		"v1alpha5 API": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"karpenter.sh/provisioner-name": "test v1alpha5",
					},
				},
			},
			Expected: "(Karpenter) test v1alpha5",
		},
		"v1beta1 API": {
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"karpenter.sh/nodepool": "test v1beta1",
					},
				},
			},
			Expected: "(Karpenter) test v1beta1",
		},
	}

	for name, tc := range testTable {
		t.Run(name, func(t *testing.T) {
			actual := findNodepool(*tc.Node, tc.CustomLabel)
			assert.Equal(t, tc.Expected, actual, "failed")
		})
	}
}
