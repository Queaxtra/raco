package clipboard

import (
	"github.com/atotto/clipboard"
)

func Copy(text string) error {
	isEmpty := text == ""
	if isEmpty {
		return nil
	}

	return clipboard.WriteAll(text)
}
