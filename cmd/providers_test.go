
package cmd

import (
	"strings"
	"testing"

	"github.com/nlink-jp/scat/internal/appcontext"
	"github.com/nlink-jp/scat/internal/config"
)

func TestGetProvider_Success(t *testing.T) {
	testCases := []string{"slack", "mock", "test"}
	ctx := appcontext.NewContext(false, false, false, "", false, nil)

	for _, providerName := range testCases {
		t.Run(providerName, func(t *testing.T) {
			p := config.Profile{Provider: providerName}
			prov, err := GetProvider(ctx, p)

			if err != nil {
				t.Errorf("Expected no error for provider '%s', but got: %v", providerName, err)
			}
			if prov == nil {
				t.Errorf("Expected a provider instance for '%s', but got nil", providerName)
			}
		})
	}
}

func TestGetProvider_UnknownProvider(t *testing.T) {
	ctx := appcontext.NewContext(false, false, false, "", false, nil)
	p := config.Profile{Provider: "unknown-provider"}

	prov, err := GetProvider(ctx, p)

	if err == nil {
		t.Fatal("Expected an error for an unknown provider, but got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("Expected error message to contain 'unknown provider', got: %v", err)
	}
	if prov != nil {
		t.Error("Expected provider instance to be nil for an unknown provider, but it was not")
	}
}
