package commands

import "github.com/Vncntvx/typush/util"

// loadUniverseIndex loads the Universe index, warning on stderr when the index in
// use is an outdated cache copy.
func loadUniverseIndex(refresh bool) (*util.UniverseIndex, error) {
	idx, err := util.LoadUniverseIndex(refresh)
	if err != nil {
		return nil, err
	}
	if staleErr := idx.StaleErr(); staleErr != nil {
		warnf("%v, falling back to cached index", staleErr)
	}
	return idx, nil
}
