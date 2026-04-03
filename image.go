package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
)

func Pull(argument string, basePath string) {
	image_url, ok := image[argument]
	if ok {
		fmt.Println("found the image ")
	} else {
		fmt.Println("the image was not found at all ")
		return
	}
	raw_image, _ := http.Get(image_url)
	gzReader, err := gzip.NewReader(raw_image.Body)
	if err != nil {
		fmt.Println("zip extraction failed ")
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		dest_path := basePath + "/images/" + argument + "/" + header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(dest_path, 0755)
		case tar.TypeReg:
			file, _ := os.Create(dest_path)
			io.Copy(file, tarReader)
			os.Chmod(dest_path, header.FileInfo().Mode())
			file.Close()
		case tar.TypeSymlink:
			os.Symlink(header.Linkname, dest_path)

		}
	}
}
