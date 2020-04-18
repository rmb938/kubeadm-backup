package gcs

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v2"
)

type blobStorageConfig struct {
	Bucket         string `yaml:"bucket"`
	ServiceAccount string `yaml:"service_account"`
}

type blobClient struct {
	config    *blobStorageConfig
	gcsClient *storage.Client
}

func NewBlobClient(ctx context.Context, rawConfig []byte) (*blobClient, error) {
	var opts []option.ClientOption

	config := &blobStorageConfig{}
	err := yaml.Unmarshal(rawConfig, config)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing gcs blob storage config")
	}

	if config.Bucket == "" {
		return nil, errors.New("missing gcs bucket name in blob storage config")
	}

	if len(config.ServiceAccount) > 0 {
		credentials, err := google.CredentialsFromJSON(ctx, []byte(config.ServiceAccount), storage.ScopeFullControl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create credentials from JSON")
		}

		opts = append(opts, option.WithCredentials(credentials))
	} else {
		credentials, err := google.FindDefaultCredentials(ctx, storage.ScopeFullControl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find default credentials")
		}

		opts = append(opts, option.WithCredentials(credentials))
	}

	opts = append(opts,
		option.WithUserAgent(fmt.Sprintf("kubeadm-backup-%s (%s)", "0.0.1", runtime.Version())))

	gcsClient, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error creating gcs client")
	}

	bc := &blobClient{
		config:    config,
		gcsClient: gcsClient,
	}
	return bc, nil
}

func (bc *blobClient) Create(ctx context.Context, objectName string, reader io.Reader) error {
	bkt := bc.gcsClient.Bucket(bc.config.Bucket)
	obj := bkt.Object(objectName)
	objWriter := obj.NewWriter(ctx)

	_, err := io.Copy(objWriter, reader)
	if err != nil {
		return errors.Wrapf(err, "error writing object %s to bucket %s", objectName, bc.config.Bucket)
	}

	err = objWriter.Close()
	if err != nil {
		return errors.Wrapf(err, "error closing object writer")
	}

	return nil
}

func (bc *blobClient) Read(ctx context.Context, objectName string) (io.Reader, error) {
	bkt := bc.gcsClient.Bucket(bc.config.Bucket)
	obj := bkt.Object(objectName)

	return obj.NewReader(ctx)
}

func (bc *blobClient) List(ctx context.Context) <-chan interface{} {
	objectNamesChan := make(chan interface{}, 1)
	bkt := bc.gcsClient.Bucket(bc.config.Bucket)

	go func(ctx context.Context, bkt *storage.BucketHandle, objectNamesChan chan<- interface{}) {
		defer close(objectNamesChan)
		it := bkt.Objects(ctx, nil)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				objectNamesChan <- err
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				objectNamesChan <- attrs.Name
			}
		}
	}(ctx, bkt, objectNamesChan)

	return objectNamesChan
}

func (bc *blobClient) Delete(ctx context.Context, objectName string) error {
	bkt := bc.gcsClient.Bucket(bc.config.Bucket)
	obj := bkt.Object(objectName)

	return obj.Delete(ctx)
}

func (bc *blobClient) Close() error {
	return bc.gcsClient.Close()
}
