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
	"strings"

	"github.com/go-kit/log"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
)

type Store struct {
	bucket       objstore.Bucket
	tsdbpath     string
	objectpath   string
	objectconfig string
	bucketlogger *slog.Logger
}

func newStore(tsdbPath, objectConfig, objectPath string, logger *slog.Logger) (*Store, error) {
	configBytes, err := os.ReadFile(objectConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if len(configBytes) == 0 {
		fmt.Println("Config file is empty, exiting container.")
		os.Exit(0)
	}

	bucket, err := client.NewBucket(log.NewNopLogger(), configBytes, "block-sync")
	if err != nil {
		return nil, fmt.Errorf("failed to create bucket existence:%w", err)
	}
	path, err := os.ReadFile(objectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read objectKey file: %w", err)
	}

	content := strings.TrimSpace(string(path))
	lines := strings.Split(content, "\n")
	var directory string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "path:") {
			directory = strings.TrimSpace(strings.TrimPrefix(line, "path:"))
			break
		}
	}

	if directory == "" {
		return nil, fmt.Errorf("expected 'path:' prefix not found")
	}

	return &Store{
		bucket:       bucket,
		tsdbpath:     tsdbPath,
		objectpath:   directory,
		objectconfig: objectConfig,
		bucketlogger: logger,
	}, nil
}

func (c *Store) upload(ctx context.Context) error {
	exists, err := c.bucket.Exists(ctx, c.objectpath)
	if err != nil {
		return fmt.Errorf("failed to check new bucket:%w", err)
	}
	c.bucketlogger.Info("Bucket checked  Successfully", "Bucket name", exists)

	err = objstore.UploadDir(ctx, log.NewNopLogger(), c.bucket, c.tsdbpath, c.objectpath)
	if err != nil {
		c.bucketlogger.Error("Failed to upload directory", "path", c.tsdbpath, "error", err)
		return fmt.Errorf("failed to upload directory from path %s to bucket: %w", c.tsdbpath, err)
	}

	c.bucketlogger.Info("Successfully uploaded directory", "path", c.tsdbpath, "bucket", c.bucket.Name())
	return nil
}

func (c *Store) download(ctx context.Context) error {
	exists, err := c.bucket.Exists(ctx, c.objectpath)
	if err != nil {
		return fmt.Errorf("failed to check new bucket:%w", err)
	}
	c.bucketlogger.Info("Bucket checked  Successfully", "Bucket name", exists)

	err = objstore.DownloadDir(ctx, log.NewNopLogger(), c.bucket, "dir/", c.objectpath, c.tsdbpath)
	if err != nil {
		c.bucketlogger.Error("Failed to download directory", "path", c.tsdbpath, "error", err)
		return fmt.Errorf("failed to download directory from path %s to bucket: %w", c.tsdbpath, err)
	}

	c.bucketlogger.Info("Successfully downloaded directory", "path", c.tsdbpath, "bucket", c.bucket.Name())
	return nil
}
