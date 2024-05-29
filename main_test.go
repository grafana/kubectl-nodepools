package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNodeCondition(t *testing.T) {
	testTable := []struct {
		Name     string
		Node     *corev1.Node
		Expected string
	}{
		{
			Name: "Basic Node",
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

	for _, tc := range testTable {
		actual := nodeCondition(*tc.Node)
		assert.Equal(t, tc.Expected, actual, "fail")
	}
}

func TestNodepoolLabels(t *testing.T) {
	testTable := []struct {
		Name        string
		Node        *corev1.Node
		CustomLabel string
		Expected    string
	}{
		{
			Name: "Basic Node - No matching label",
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"custom-label": "custom label",
					},
				},
			},
			Expected: "-",
		},
		{
			Name: "Basic Node - Custom Label",
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
		{
			Name: "Basic Node - AWS",
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"eks.amazonaws.com/nodegroup": "test AWS",
					},
				},
			},
			Expected: "test AWS",
		},
		{
			Name: "Basic Node - GCP",
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"cloud.google.com/gke-nodepool": "test GCP",
					},
				},
			},
			Expected: "test GCP",
		},
		{
			Name: "Basic Node - AKS",
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"kubernetes.azure.com/agentpool": "test AKS",
					},
				},
			},
			Expected: "test AKS",
		},
		{
			Name: "Basic Node - DigitalOcean",
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"doks.digitalocean.com/node-pool-id": "test DigitalOcean",
					},
				},
			},
			Expected: "test DigitalOcean",
		},
		{
			Name: "Basic Node, v1alpha5",
			Node: &corev1.Node{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"karpenter.sh/provisioner-name": "test v1alpha5",
					},
				},
			},
			Expected: "(Karpenter) test v1alpha5",
		},
		{
			Name: "Basic Node, v1beta1",
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

	for _, tc := range testTable {
		actual := findNodepool(*tc.Node, tc.CustomLabel)
		assert.Equal(t, tc.Expected, actual, "fail")
	}
}
