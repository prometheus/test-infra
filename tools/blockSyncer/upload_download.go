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

	"github.com/go-kit/log"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Store struct {
	Bucket       objstore.Bucket
	TsdbPath     string
	ObjectKey    string
	ObjectConfig string
}

func newstore(tsdbPath, objectConfig, objectKey string) *Store {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	configBytes, err := os.ReadFile(objectConfig)
	if err != nil {
		logger.Error("failed to read config file", "error", err)
		return nil
	}

	bucket, err := client.NewBucket(log.NewNopLogger(), configBytes, "block-sync")
	if err != nil {
		logger.Error("Failed to create bucket existence", "error", err)
		return nil
	}

	exists, err := bucket.Exists(context.Background(), objectKey)
	if err != nil {
		logger.Error("Failed to create new bucket", "error", err)
		return nil
	}
	logger.Info("Bucket existence check", "bucket_name", objectKey, "exists", exists)

	return &Store{
		Bucket:       bucket,
		TsdbPath:     tsdbPath,
		ObjectConfig: objectConfig,
		ObjectKey:    objectKey,
	}
}

func (c *Store) upload(*kingpin.ParseContext) error {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := objstore.UploadDir(context.Background(), log.NewNopLogger(), c.Bucket, c.TsdbPath, c.ObjectKey)
	if err != nil {
		logger.Error("Failed to upload directory", "path", c.TsdbPath, "error", err)
		return fmt.Errorf("failed to upload directory from path %s to bucket: %v", c.TsdbPath, err)
	}

	logger.Info("Successfully uploaded directory", "path", c.TsdbPath, "bucket", c.Bucket.Name())
	return nil
}

func (c *Store) download(*kingpin.ParseContext) error {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	err := objstore.DownloadDir(context.Background(), log.NewNopLogger(), c.Bucket, "dir/", c.ObjectKey, c.TsdbPath)
	if err != nil {
		logger.Error("Failed to download directory", "path", c.TsdbPath, "error", err)
		return fmt.Errorf("failed to download directory from path %s to bucket: %v", c.TsdbPath, err)
	}

	logger.Info("Successfully downloaded directory", "path", c.TsdbPath, "bucket", c.Bucket.Name())
	return nil
}
