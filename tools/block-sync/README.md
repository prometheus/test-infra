# block-sync - TSDB Data Synchronization Tool

The `block-sync` command is a CLI tool designed to synchronize TSDB data with an object storage system. 

## Table of Contents

1. [Upload](#upload)
2. [Download](#download)

## Command Flags

- ``` -h , --help```:Displays context-sensitive help 
- ``` - tsdb-path```: Path for The TSDB data in prometheus
- ```- objstore.config-file```: Path for The Config file
- ```- path```: Path within the objectstorage where to store block data , i.e a Directory.
## Note  
> Use the `path` flag with a file specifying the path: *your directory name*.

## Upload

The `upload` command allows you to upload TSDB data from a specified path to an object storage bucket. This command is essential for backing up your TSDB data or migrating it to an object storage solution for future use.

### Usage

```bash
./block-sync upload --tsdb-path=<path-to-tsdb> --objstore.config-file=<path-to-config> --path=<path-to-bucket-config-file>


```
## Download

The `download` command allows you to retrieve TSDB data from an object storage bucket to a specified local path. This command is essential for restoring your TSDB data or retrieving it for local analysis and processing.

### Usage

```bash
./block-sync download --tsdb-path=<path-to-tsdb> --objstore.config-file=<path-to-config> --path=<path-to-bucket-config-file>
```
## Config File

The configuration file is essential for connecting to your object storage solution. Below are basic templates for different object storage systems.

```yaml
type: s3, GCS , AZURE , etc.
config:
  bucket: your-bucket-name
  endpoint: https://your-endpoint
  access_key: your-access-key
  secret_key: your-secret-key
  insecure: false  # Set to true if using HTTP instead of HTTPS
```
You can customize the config file ,  follow this link [Storage.md](https://thanos.io/tip/thanos/storage.md/)

