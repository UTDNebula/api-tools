package utils

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/chromedp/cdproto/network"
)

// TestPrivateNetworkRequestPolicyUnmarshal verifies newer CDP enum values
// emitted by Chrome can be decoded without protocol unmarshal errors.
func TestPrivateNetworkRequestPolicyUnmarshal(t *testing.T) {
	t.Parallel()

	testCases := []network.PrivateNetworkRequestPolicy{
		network.PrivateNetworkRequestPolicyAllow,
		network.PrivateNetworkRequestPolicyPreflightBlock,
		network.PrivateNetworkRequestPolicyPermissionBlock,
		network.PrivateNetworkRequestPolicyPermissionWarn,
	}

	for _, testCase := range testCases {
		t.Run(testCase.String(), func(t *testing.T) {
			t.Parallel()

			payload := fmt.Sprintf(`"%s"`, testCase)
			var result network.PrivateNetworkRequestPolicy
			if err := json.Unmarshal([]byte(payload), &result); err != nil {
				t.Fatalf("failed to unmarshal %s: %v", testCase, err)
			}

			if result != testCase {
				t.Fatalf("expected %s, got %s", testCase, result)
			}
		})
	}
}
