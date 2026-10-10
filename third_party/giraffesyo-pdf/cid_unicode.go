package pdf

import (
	"embed"
	"sync"
)

// Official Adobe character collections recover CID-keyed fonts that omit
// ToUnicode. They are compiled in; extraction never fetches external resources.
//
//go:embed cmaps/Adobe-*-UCS2
var cidMaps embed.FS

var cidMapCache sync.Map

type cidMapEntry struct {
	once sync.Once
	data *cmapData
}

func adobeCIDUnicode(registry, ordering string) *cmapData {
	if registry != "Adobe" {
		return nil
	}
	switch ordering {
	case "GB1", "CNS1", "Japan1", "Korea1":
	default:
		return nil
	}
	cached, _ := cidMapCache.LoadOrStore(ordering, &cidMapEntry{})
	entry := cached.(*cidMapEntry)
	entry.once.Do(func() {
		data, err := cidMaps.ReadFile("cmaps/Adobe-" + ordering + "-UCS2")
		if err != nil {
			return
		}
		entry.data, _ = parseCMap(data, nil, map[string]bool{})
	})
	return entry.data
}
