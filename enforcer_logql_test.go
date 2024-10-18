package main

import (
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestLogqlEnforcer(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		tenantLabels   LabelType
		expectedResult string
		expectErr      bool
	}{
		{
			name:           "Valid query and tenant labels",
			query:          "{kubernetes_namespace_name=\"test\"}",
			tenantLabels:   LabelType{"test": true},
			expectedResult: "{kubernetes_namespace_name=\"test\"}",
			expectErr:      false,
		},
		{
			name:           "Empty query and valid tenant labels",
			query:          "",
			tenantLabels:   LabelType{"test": true},
			expectedResult: "{kubernetes_namespace_name=\"test\"}",
			expectErr:      false,
		},
		{
			name:         "Valid query and invalid tenant labels",
			query:        "{kubernetes_namespace_name=\"test\"}",
			tenantLabels: LabelType{"invalid": true},
			expectErr:    true,
		},
	}

	enforcer := LogQLEnforcer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := enforcer.Enforce(tt.query, tt.tenantLabels, "kubernetes_namespace_name")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestMatchNamespaceMatchers(t *testing.T) {
	tests := []struct {
		name         string
		matchers     []*labels.Matcher
		tenantLabels LabelType
		expectErr    bool
	}{
		{
			name: "Valid matchers and tenant labels",
			matchers: []*labels.Matcher{
				{
					Type:  labels.MatchEqual,
					Name:  "kubernetes_namespace_name",
					Value: "test",
				},
			},
			tenantLabels: LabelType{"test": true},
			expectErr:    false,
		},
		{
			name: "Invalid matchers and valid tenant labels",
			matchers: []*labels.Matcher{
				{
					Type:  labels.MatchEqual,
					Name:  "kubernetes_namespace_name",
					Value: "invalid",
				},
			},
			tenantLabels: LabelType{"test": true},
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := matchNamespaceMatchers(tt.matchers, tt.tenantLabels, "kubernetes_namespace_name")
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
