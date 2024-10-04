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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Tool for storing TSDB data to object storage")
	app.HelpFlag.Short('h')

	var (
		s            *Store
		tsdbPath     string
		objectConfig string
		objectKey    string
		logger       *slog.Logger
	)
	objstore := app.Command("blocksync", `Using an object storage to store the data`)
	objstore.Flag("tsdb-path", "Path for The TSDB data in prometheus").Required().StringVar(&tsdbPath)
	objstore.Flag("objstore.config-file", "Path for The Config file").Required().StringVar(&objectConfig)
	objstore.Flag("key", "Path for the Key where to store block data").Required().StringVar(&objectKey)
	objstore.Action(func(c *kingpin.ParseContext) error {
		var err error
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
		s, err = newStore(tsdbPath, objectConfig, objectKey, logger)
		if err != nil {
			logger.Error("Failed to create store", "error", err)
			return fmt.Errorf("fail to create store :%w", err)
		}
		return nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uploadCmd := objstore.Command("upload", "Uploading data to objstore")
	uploadCmd.Action(func(c *kingpin.ParseContext) error {
		return s.upload(ctx)
	})

	downloadCmd := objstore.Command("download", "Downloading data from objstore")
	downloadCmd.Action(func(c *kingpin.ParseContext) error {
		return s.download(ctx)
	})
	kingpin.MustParse(app.Parse(os.Args[1:]))

}
