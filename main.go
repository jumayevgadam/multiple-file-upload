package main

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/chai2010/webp"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/upload", uploadHandler)

	e.Start(":8083")
}

func uploadHandler(c echo.Context) error {
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	defer form.RemoveAll()

	files := form.File["files"]
	if len(files) == 0 {
		return c.String(http.StatusBadRequest, "No files uploaded")
	}

	uploadDir := filepath.Join(".", "uploads")
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return err
	}

	var g errgroup.Group
	var mu sync.Mutex
	results := make([]string, len(files))

	for i, fileHeader := range files {
		i, fileHeader := i, fileHeader

		g.Go(func() error {
			src, err := fileHeader.Open()
			if err != nil {
				return err
			}
			defer src.Close()

			originalPath := filepath.Join(uploadDir, fileHeader.Filename)
			dst, err := os.Create(originalPath)
			if err != nil {
				return err
			}
			defer dst.Close()

			if _, err := io.Copy(dst, src); err != nil {
				return err
			}

			webpPath := originalPath + ".webp"
			err = convertToWebP(originalPath, webpPath)
			if err != nil {
				return err
			}

			mu.Lock()
			results[i] = webpPath
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Files uploaded and converted successfully",
		"results": results,
	})
}

func convertToWebP(inputPath, outputPath string) error {
	// Open the input file
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Encode as WebP with reasonable quality
	err = webp.Encode(out, img, &webp.Options{Quality: 80})
	if err != nil {
		return err
	}

	return nil
}
