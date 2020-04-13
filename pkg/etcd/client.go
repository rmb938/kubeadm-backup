package etcd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
)

type Client struct {
	clientv3Client *clientv3.Client
}

func NewEtcdClient(endpoint, caFile, keyFile, certFile string) (*Client, error) {
	tlsConfig, err := loadCertificates(caFile, keyFile, certFile)
	if err != nil {
		return nil, errors.Wrap(err, "error loading etcd certificates")
	}

	clientv3Client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{endpoint},
		TLS:       tlsConfig,
	})
	if err != nil {
		return nil, errors.Wrap(err, "error creating etcd client")
	}

	client := &Client{
		clientv3Client: clientv3Client,
	}
	return client, nil
}

func loadCertificates(caFile, keyFile, certFile string) (*tls.Config, error) {
	cfg := &tls.Config{}

	if caFile != "" {
		caPEM, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}

		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(caPEM)
		if !ok {
			return nil, fmt.Errorf("failed to add etcd ca certificate")
		}

		cfg.RootCAs = certPool
	}

	if keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}

		cfg.Certificates = []tls.Certificate{cert}
	}

	return cfg, nil
}

func (c *Client) Snapshot(ctx context.Context) (io.ReadCloser, error) {
	return c.clientv3Client.Snapshot(ctx)
}

func (c *Client) Sync(ctx context.Context) error {
	return c.clientv3Client.Sync(ctx)
}

func (c *Client) Close() error {
	return c.clientv3Client.Close()
}
