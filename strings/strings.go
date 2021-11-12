package strings

import "fmt"

const (
	_ = 1 << (10 * iota)
	KB
	MB
	GB
)

func FormatBytes(size int) string {
	var f float32 = float32(size)
	u := "B"
	switch {
	case size >= GB:
		f, u = f/GB, "GB"
	case size >= MB:
		f, u = f/MB, "MB"
	case size >= KB:
		f, u = f/KB, "KB"
	}
	return fmt.Sprintf("%.1f%s", f, u)
}
