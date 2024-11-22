// Copyright 2024 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
)

func main() {
	var (
		tsdbPath     string
		objectConfig string
		objectPath   string
	)
	uploadCmd := flag.NewFlagSet("upload", flag.ExitOnError)
	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)

	uploadCmd.StringVar(&tsdbPath, "tsdb-path", "", "Uploading data to objstore")
	uploadCmd.StringVar(&objectConfig, "objstore.config-file", "", "Path for The Config file")
	uploadCmd.StringVar(&objectPath, "path", "", "Path within the objectstorage")

	downloadCmd.StringVar(&tsdbPath, "tsdb-path", "", "Downloading data to objstore")
	downloadCmd.StringVar(&objectConfig, "objstore.config-file", "", "Path for The Config file")
	downloadCmd.StringVar(&objectPath, "path", "", "Path within the objectstorage")
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Println("  upload     Uploads data to the object store")
		fmt.Println("  download   Downloads data from the object store")
		fmt.Println("Flags:")
		fmt.Println("  --tsdb-path               Path to TSDB data")
		fmt.Println("  --objstore.config-file    Path to the object store config file")
		fmt.Println("  --path                    Directory in objectstorage for uploading or downloading data")
		fmt.Println()
	}

	if len(os.Args) < 2 {
		logger.Error("Expected 'upload' or 'download' subcommands")
		flag.Usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "upload":
		if err := uploadCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing upload command:", err)
			os.Exit(1)
		}
	case "download":
		if err := downloadCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println("Error parsing download command:", err)
			os.Exit(1)
		}
	default:
		logger.Error("Expected 'upload' or 'download' subcommands")
		flag.Usage()
		os.Exit(1)
	}

	if tsdbPath == "" || objectConfig == "" || objectPath == "" {
		fmt.Println("error: all flags --tsdb-path, --objstore.config-file, and --path are required.")
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := newStore(tsdbPath, objectConfig, objectPath, logger)
	if err != nil {
		logger.Error("Failed to create store", "error", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "upload":
		err = store.upload(ctx)
		if err != nil {
			logger.Error("Failed to upload data", "Error", err)
			os.Exit(1)
		}
	case "download":
		err = store.download(ctx)
		if err != nil {
			logger.Error("Failed to download data", "error", err)
			os.Exit(1)
		}
	}
}
