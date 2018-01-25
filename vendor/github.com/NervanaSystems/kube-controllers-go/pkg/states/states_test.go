package states

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsTerminal(t *testing.T) {
	testCases := []struct {
		st       State
		expected bool
	}{
		{
			st:       Completed,
			expected: true,
		},
		{
			st:       Failed,
			expected: true,
		},
		{
			st:       Pending,
			expected: false,
		},
		{
			st:       Running,
			expected: false,
		},
	}

	for _, testCase := range testCases {
		actual := IsTerminal(testCase.st)
		require.Equal(t, actual, testCase.expected)
	}
}

func TestIsOneOf(t *testing.T) {
	testCases := []struct {
		currentState State
		targetStates []State
		expected     bool
	}{
		{
			currentState: Pending,
			targetStates: []State{Pending},
			expected:     true,
		},
		{
			currentState: Pending,
			targetStates: []State{Pending, Failed},
			expected:     true,
		},
		{
			currentState: Pending,
			targetStates: []State{Failed, Pending},
			expected:     true,
		},
		{
			currentState: Pending,
			targetStates: []State{Running},
			expected:     false,
		},
		{
			currentState: Pending,
			targetStates: []State{Failed, Running},
			expected:     false,
		},
	}

	for _, testCase := range testCases {
		actual := testCase.currentState.IsOneOf(testCase.targetStates...)
		require.Equal(t, actual, testCase.expected)
	}
}
