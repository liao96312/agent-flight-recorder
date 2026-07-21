//go:build !windows

package afr

import "os"

func isLinkLike(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }
