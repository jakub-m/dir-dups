package cleanup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/assert"
)

func TestParseManifestLines(t *testing.T) {
	type tct struct {
		in string
		ok bool
	}
	for _, tc := range []tct{
		{
			in: `#`,
			ok: false,
		},
		{
			in: `keep	foo	bar`,
			ok: true,
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			_, err := ParseLineToManifestEntry(tc.in)
			if tc.ok {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
