package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLabelsCM(t *testing.T) {
	cmh := ConfigMapHandler{
		labels: ACLs{
			"user1":      {"namespace": {"u1": true, "u2": true}},
			"user2":      {"namespace": {"u3": true, "u4": true}},
			"group1":     {"namespace": {"g1": true, "g2": true}},
			"group2":     {"namespace": {"g3": true, "g4": true}},
			"adminGroup": {"namespace": {"#cluster-wide": true, "g4": true}},
		},
	}

	cases := []struct {
		name     string
		username string
		groups   []string
		expected Filter
		skip     bool
	}{
		{
			name:     "User with groups",
			username: "user1",
			groups:   []string{"group1", "group2"},
			expected: Filter{
				"namespace": {
					"u1": true,
					"u2": true,
					"g1": true,
					"g2": true,
					"g3": true,
					"g4": true,
				},
			},
		},
		{
			name:     "User without groups",
			username: "user2",
			groups:   []string{},
			expected: Filter{
				"namespace": {
					"u3": true,
					"u4": true,
				},
			},
		},
		{
			name:     "Non-existent user",
			username: "user3",
			groups:   []string{"group1"},
			expected: Filter{
				"namespace": {
					"g1": true,
					"g2": true,
				},
			},
		},
		{
			name:     "Non-existent group",
			username: "user1",
			groups:   []string{"group3"},
			expected: Filter{
				"namespace": {
					"u1": true,
					"u2": true,
				},
			},
		},
		{
			name:     "admin_group",
			username: "blubb",
			groups:   []string{"adminGroup"},
			expected: nil,
			skip:     true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			labels, skip := cmh.GetLabels(OAuthToken{PreferredUsername: tc.username, Groups: tc.groups})
			assert.Equal(t, tc.expected, labels)
			assert.Equal(t, tc.skip, skip)
		})
	}
}
