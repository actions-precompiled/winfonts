package winfonts

import (
	"context"
	"fmt"
	"io"
	"iter"
	"log"
	"os"
	"path/filepath"

	"github.com/Microsoft/go-winio/wim"
	"github.com/Xmister/udf"
)

type FontExtractor struct {
	iso    *udf.Udf
	output string
}

func NewFontExtractor(ra io.ReaderAt, output string) (*FontExtractor, error) {
	iso, err := udf.NewUdfFromReader(ra)
	if err != nil {
		return nil, err
	}
	return &FontExtractor{
		iso:    iso,
		output: output,
	}, nil
}

func (e *FontExtractor) saveReader(ctx context.Context, r io.Reader, outputFile string) error {
	log.Printf("  Extracting font: %s", outputFile)
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

func (e *FontExtractor) wimFiles(image *wim.Image) iter.Seq2[*wim.File, error] {
	return func(yield func(*wim.File, error) bool) {
		var walk func(*wim.File) bool

		walk = func(dir *wim.File) bool {
			entries, err := dir.Readdir()
			if err != nil {
				yield(nil, fmt.Errorf("failed to read directory: %w", err))
				return false
			}

			for _, entry := range entries {

				if !yield(entry, nil) {
					return false
				}

				if entry.IsDir() {
					if !walk(entry) {
						return false
					}
				}
			}
			return true
		}

		root, err := image.Open()
		if err != nil {
			yield(nil, fmt.Errorf("failed to open WIM image: %w", err))
			return
		}

		walk(root)
	}
}

func (e *FontExtractor) handleWimImage(ctx context.Context, image *wim.Image) error {
	for file, err := range e.wimFiles(image) {
		if err != nil {
			return err
		}

		log.Printf("wimfile: %s", file.Name)
		ext := filepath.Ext(file.Name)
		if ext == ".ttf" {
			r, err := file.Open()
			if err != nil {
				log.Printf("failed to open font file %s in WIM image: %v", file.Name, err)
				continue
			}
			err = e.saveReader(ctx, r, file.Name)
			if err != nil {
				log.Printf("failed to save font %s: %v", file.Name, err)
				continue
			}
		}
	}
	return nil
}

func (e *FontExtractor) handleWim(ctx context.Context, f udf.File) error {
	wimName := f.Name()
	log.Printf("Processing WIM file: %s", wimName)
	r := f.NewReader()
	bundle, err := wim.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to read WIM file %s: %w", wimName, err)
	}
	log.Printf("  Found %d image(s) in WIM", len(bundle.Image))
	for idx, image := range bundle.Image {
		log.Printf("wimimage: %s", image.Name)
		log.Printf("  Processing image %d/%d", idx+1, len(bundle.Image))
		err = e.handleWimImage(ctx, image)
		if err != nil {
			return fmt.Errorf("failed to process image in WIM file %s: %w", wimName, err)
		}
	}
	return nil
}

func (e *FontExtractor) isoFiles(yield func(udf.File) bool) {
	var walk func([]udf.File) bool

	walk = func(files []udf.File) bool {
		for _, item := range files {
			if !yield(item) {
				return false
			}

			if item.IsDir() {
				children := e.iso.ReadDir(item.FileEntry())
				if !walk(children) {
					return false
				}
			}
		}
		return true
	}

	walk(e.iso.ReadDir(nil))
}

func (e *FontExtractor) extractFonts(ctx context.Context) error {
	log.Printf("Starting font extraction from ISO")
	log.Printf("Scanning ISO for WIM files...")
	for item := range e.isoFiles {
		log.Printf("isofile: %s %s", item.Name(), filepath.Ext(item.Name()))
		if filepath.Ext(item.Name()) == ".wim" {
			err := e.handleWim(ctx, item)
			if err != nil {
				return fmt.Errorf("failed to extract fonts from %s: %w", item.Name(), err)
			}
		}
	}
	log.Printf("Font extraction completed successfully")
	return nil
}

func (e *FontExtractor) Run(ctx context.Context) error {
	return e.extractFonts(ctx)
}
