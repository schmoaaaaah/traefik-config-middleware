package aggregator

// ShouldIgnoreRouter checks if a router should be ignored based on its entrypoints.
// Returns true if any of the router's entrypoints are in the ignore list.
func ShouldIgnoreRouter(router TraefikRouter, ignoreEntryPoints []string) bool {
	if len(ignoreEntryPoints) == 0 {
		return false
	}

	// Check if any of the router's entrypoints are in the ignore list
	for _, routerEP := range router.EntryPoints {
		for _, ignoreEP := range ignoreEntryPoints {
			if routerEP == ignoreEP {
				return true
			}
		}
	}

	return false
}
