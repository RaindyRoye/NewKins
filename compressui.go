//go:build ignore

package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gokins/core/utils"
)

var pth string

func main() {
	flag.StringVar(&pth, "d", "", "ui dir")
	flag.Parse()
	if pth == "" {
		pth = "../web/dist"
	}
	err := gengo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bdzip error: %v\n", err)
		os.Exit(1)
	}
}

func gengo() error {
	zipfl := filepath.Join(utils.HomePath(), "dist.zip")
	if err := os.RemoveAll(zipfl); err != nil {
		return fmt.Errorf("remove old zip %s: %w", zipfl, err)
	}
	defer func() { _ = os.RemoveAll(zipfl) }()
	if err := utils.Zip(pth, zipfl, true); err != nil {
		return fmt.Errorf("zip directory %s: %w", pth, err)
	}
	bts, err := os.ReadFile(zipfl)
	if err != nil {
		return fmt.Errorf("read zip file %s: %w", zipfl, err)
	}
	cont := base64.StdEncoding.EncodeToString(bts)
	if err := os.WriteFile("comm/uis.go",
		[]byte(fmt.Sprintf("package comm\n\nconst StaticPkg = %q", cont)),
		0644); err != nil {
		return fmt.Errorf("write comm/uis.go: %w", err)
	}
	fmt.Println("ui insert go ok!!!")
	return nil
}
