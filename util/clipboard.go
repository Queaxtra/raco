package util

import (
	"raco/util/func/clipboard"
)

func CopyToClipboard(text string) error {
	return clipboard.Copy(text)
}
