// +build k8srequired

package appcatalogentry

import (
	"context"
	"testing"
)

// TestAppCatalogEntry checks that appcatalogentry CRs and the metadata they
// contain are generated correctly for an appcatalog CR.
func TestAppCatalogEntry(t *testing.T) {
	ctx := context.Background()

	config.Logger.LogCtx(ctx, "level", "debug", "message", "add test logic")
}
