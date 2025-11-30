package winfonts

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Microsoft/go-winio/wim"
	"github.com/kdomanski/iso9660"
)

type FontExtractor struct {
	iso    *iso9660.Image
	output string
}

func NewFontExtractor(ra io.ReaderAt, output string) (*FontExtractor, error) {
	iso, err := iso9660.OpenImage(ra)
	if err != nil {
		return nil, err
	}
	return &FontExtractor{
		iso:    iso,
		output: output,
	}, nil
}

func (e *FontExtractor) saveReader(ctx context.Context, r io.Reader, outputFile string) error {
	location := filepath.Join(e.output, outputFile)
	f, err := os.Create(location)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputFile, err)
	}
	buf := make([]byte, 64*1024)
	_, err = io.CopyBuffer(f, r, buf)
	if err != nil {
		return fmt.Errorf("failed to copy data to file %s: %w", outputFile, err)
	}
	return nil
}

func (e *FontExtractor) handleWimImage(ctx context.Context, image *wim.Image) error {
	f, err := image.Open()
	if err != nil {
		return fmt.Errorf("failed to open WIM image: %w", err)
	}
	dir, err := f.Readdir()
	if err != nil {
		return fmt.Errorf("failed to read WIM image directory: %w", err)
	}
	for _, item := range dir {
		ext := filepath.Ext(item.Name)
		if ext == "ttf" {
			f, err := item.Open()
			if err != nil {
				return fmt.Errorf("failed to open font file %s in WIM image: %w", item.Name, err)
			}
			err = e.saveReader(ctx, f, item.Name)
			if err != nil {
				return fmt.Errorf("failed to save font %s: %w", item.Name, err)
			}
		}
	}
	return nil
}

func (e *FontExtractor) handleWim(ctx context.Context, f *iso9660.File) error {
	wimName := f.Name()
	r := f.Reader().(*io.SectionReader)
	bundle, err := wim.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to read WIM file %s: %w", wimName, err)
	}
	for _, image := range bundle.Image {
		err = e.handleWimImage(ctx, image)
		if err != nil {
			return fmt.Errorf("failed to process image in WIM file %s: %w", wimName, err)
		}
	}
	return nil
}

func (e *FontExtractor) extractFonts(ctx context.Context) error {
	f, err := e.iso.RootDir()
	if err != nil {
		return fmt.Errorf("failed to access ISO root directory: %w", err)
	}
	children, err := f.GetAllChildren()
	if err != nil {
		return fmt.Errorf("failed to list ISO contents: %w", err)
	}
	for _, item := range children {
		if strings.HasSuffix(item.Name(), ".wim") {
			err := e.handleWim(ctx, item)
			if err != nil {
				return fmt.Errorf("failed to extract fonts from %s: %w", item.Name(), err)
			}
		}
	}
	return nil
}

func (e *FontExtractor) Run(ctx context.Context) error {
	return e.extractFonts(ctx)
}
