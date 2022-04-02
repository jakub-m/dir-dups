package cleanup

type Manifest []ManifestEntry

type ManifestEntry struct {
	Operation ManifestOperation
	Hash      string
	Path      string
}

type ManifestOperation string

const (
	Keep ManifestOperation = "keep"
	Move                   = "move"
)
