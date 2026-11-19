package version

import (
	"os"
	"strings"
)

// Number is the product version shown in API and UI.
// Prefer env NINGPM_VERSION; fall back to build-time ldflags or default.
var Number = "0.25.41"

func init() {
	if v := strings.TrimSpace(os.Getenv("NINGPM_VERSION")); v != "" {
		Number = v
	}
}
