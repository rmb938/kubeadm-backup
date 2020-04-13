package s3

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var defaultConfig = blobStorageConfig{
	HTTPConfig: httpConfig{
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 2 * time.Minute,
	},
}

type httpConfig struct {
	IdleConnTimeout       time.Duration `yaml:"idle_conn_timeout"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout"`
	InsecureSkipVerify    bool          `yaml:"insecure_skip_verify"`
}

type blobStorageConfig struct {
	Endpoint  string `yaml:"endpoint"`
	Insecure  bool   `yaml:"insecure"`
	Bucket    string `yaml:"bucket"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`

	HTTPConfig httpConfig `json:"http_config"`
}

type blobClient struct {
	config      *blobStorageConfig
	minioClient *minio.Client
}

func NewBlobClient(rawConfig []byte) (*blobClient, error) {
	config := &defaultConfig
	err := yaml.Unmarshal(rawConfig, config)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing s3 blob storage config")
	}

	var chain []credentials.Provider

	if config.AccessKey != "" {
		chain = []credentials.Provider{&credentials.Static{
			Value: credentials.Value{
				AccessKeyID:     config.AccessKey,
				SecretAccessKey: config.SecretKey,
				SignerType:      credentials.SignatureV4,
			},
		}}
	} else {
		chain = []credentials.Provider{
			&credentials.EnvAWS{},
			&credentials.FileAWSCredentials{},
			&credentials.IAM{
				Client: &http.Client{
					Transport: http.DefaultTransport,
				},
			},
		}
	}

	minioClient, err := minio.NewWithCredentials(config.Endpoint, credentials.NewChainCredentials(chain), !config.Insecure, "")
	if err != nil {
		return nil, errors.Wrap(err, "error creating s3 client")
	}

	minioClient.SetAppInfo("kube-baremetal", fmt.Sprintf("%s (%s)", "0.0.1", runtime.Version()))
	minioClient.SetCustomTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       config.HTTPConfig.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// The ResponseHeaderTimeout here is the only change
		// from the default minio transport, it was introduced
		// to cover cases where the tcp connection works but
		// the server never answers. Defaults to 2 minutes.
		ResponseHeaderTimeout: config.HTTPConfig.ResponseHeaderTimeout,
		// Set this value so that the underlying transport round-tripper
		// doesn't try to auto decode the body of objects with
		// content-encoding set to `gzip`.
		//
		// Refer: https://golang.org/src/net/http/transport.go?h=roundTrip#L1843.
		DisableCompression: true,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: config.HTTPConfig.InsecureSkipVerify},
	})

	return &blobClient{
		config:      config,
		minioClient: minioClient,
	}, nil
}

func (b *blobClient) Create(ctx context.Context, objectName string, reader io.Reader) error {
	_, err := b.minioClient.PutObjectWithContext(ctx, b.config.Bucket, objectName, reader, -1, minio.PutObjectOptions{})
	return err
}

func (b *blobClient) Read(ctx context.Context, objectName string) (io.Reader, error) {
	return b.minioClient.GetObjectWithContext(ctx, b.config.Bucket, objectName, minio.GetObjectOptions{})
}

func (b *blobClient) List(ctx context.Context) <-chan interface{} {
	objectNamesChan := make(chan interface{}, 1)

	go func(ctx context.Context, objectNamesChan chan<- interface{}) {
		defer close(objectNamesChan)
		objectCh := b.minioClient.ListObjectsV2(b.config.Bucket, "", false, ctx.Done())
		for obj := range objectCh {
			if obj.Err != nil {
				objectNamesChan <- obj.Err
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
				objectNamesChan <- obj.Key
			}
		}
	}(ctx, objectNamesChan)

	return objectNamesChan
}

func (b *blobClient) Delete(ctx context.Context, objectName string) error {
	objectsCh := make(chan string, 1)

	go func() {
		defer close(objectsCh)
		objectsCh <- objectName
	}()

	for err := range b.minioClient.RemoveObjectsWithContext(ctx, b.config.Bucket, objectsCh) {
		return err.Err
	}

	return nil
}

func (b *blobClient) Close() error {
	panic("implement me")
}
