package blob

import (
	"context"
	"io"
)

type BlobClient interface {
	Create(ctx context.Context, objectName string, reader io.Reader) error
	Read(ctx context.Context, objectName string) (io.Reader, error)
	List(ctx context.Context) <-chan interface{}
	Delete(ctx context.Context, objectName string) error
	Close() error
}
