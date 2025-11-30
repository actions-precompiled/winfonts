package winfonts

import (
	"fmt"
	"io"

	"github.com/Microsoft/go-winio/wim"
	"github.com/kdomanski/iso9660"
)

type FontExtractor struct {
	iso *iso9660.Image
}

func NewFontExtractor(ra io.ReaderAt) (*FontExtractor, error) {
	iso, err := iso9660.OpenImage(ra)
	if err != nil {
		return nil, err
	}
	return &FontExtractor{
		iso: iso,
	}, nil
}

func (e *FontExtractor) extractFonts() error {
	f, err := e.iso.RootDir()
	if err != nil {
		return err
	}
	children, err := f.GetAllChildren()
	if err != nil {
		return err
	}
	for _, item := range children {
		fmt.Printf("%v", item)
	}
	return nil
}

func (e *FontExtractor) Extract(outputDirectory string) {
	e.extractFonts()
}
