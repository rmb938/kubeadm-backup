package blob

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/rmb938/kubeadm-backup/pkg/blob/gcs"
	"github.com/rmb938/kubeadm-backup/pkg/blob/s3"
)

type BlobStorageType string

const (
	GCS BlobStorageType = "GCS"
	S3  BlobStorageType = "S3"
)

type BlobStorageConfig struct {
	Type   BlobStorageType `yaml:"type"`
	Config interface{}     `yaml:"config"`
}

func CreateBlobClientFromConfig(configFilePath string) (BlobClient, error) {
	blobStorageConfig := &BlobStorageConfig{}

	rawConfig, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading blob storage config file %s: %w", configFilePath, err)
	}

	err = yaml.UnmarshalStrict(rawConfig, blobStorageConfig)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling blob storage config: %w", err)
	}

	config, err := yaml.Marshal(blobStorageConfig.Config)
	if err != nil {
		return nil, fmt.Errorf("error marshaling content of blob storage config: %w", err)
	}

	var client BlobClient
	switch strings.ToUpper(string(blobStorageConfig.Type)) {
	case string(GCS):
		client, err = gcs.NewBlobClient(context.Background(), config)
	case string(S3):
		client, err = s3.NewBlobClient(config)
	default:
		return nil, fmt.Errorf("blob storage config with type %s not supported: %w", blobStorageConfig.Type, err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create blob client %s: %w", blobStorageConfig.Type, err)
	}

	return client, nil
}
