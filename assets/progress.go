package assets

// Progress reports web asset pack loading to the page. Embedded builds never
// emit it.
type Progress struct {
	Pack        string
	Bytes       int // pack size once loaded
	LoadedBytes int // all loaded packs
	TotalBytes  int // every pack in the manifest
	LoadedPacks int
	TotalPacks  int
	Err         error
}
