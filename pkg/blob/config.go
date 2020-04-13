package blob

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
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

	rawConfig, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading blob storage config file %s", configFilePath)
	}

	err = yaml.UnmarshalStrict(rawConfig, blobStorageConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling blob storage config")
	}

	config, err := yaml.Marshal(blobStorageConfig.Config)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling content of blob storage config")
	}

	var client BlobClient
	switch strings.ToUpper(string(blobStorageConfig.Type)) {
	case string(GCS):
		client, err = gcs.NewBlobClient(context.Background(), config)
	case string(S3):
		client, err = s3.NewBlobClient(config)
	default:
		return nil, errors.Errorf("blob storage config with type %s not supported", blobStorageConfig.Type)
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create blob client %s", blobStorageConfig.Type)
	}

	return client, nil
}
