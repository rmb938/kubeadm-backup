package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/rmb938/kubeadm-backup/pkg/blob"
	"github.com/rmb938/kubeadm-backup/pkg/etcd"
)

var pkiFiles = []string{
	"ca.crt",
	"ca.key",
	"front-proxy-ca.crt",
	"front-proxy-ca.key",
	"sa.key",
	"sa.pub",
	path.Join("etcd", "ca.crt"),
	path.Join("etcd", "ca.key"),
}

type backup struct {
	blobClient blob.BlobClient
	etcdClient *etcd.Client

	kubeadmPKIDirectory string
}

func (b *backup) Take() error {

	// sync etcd endpoints
	syncCTX, syncCTXCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer syncCTXCancel()
	err := b.etcdClient.Sync(syncCTX)
	if err != nil {
		return fmt.Errorf("error syncing etcd endpoints: %w", err)
	}

	// take etcd snapshot
	snapshotCTX, snapshotCTXCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer snapshotCTXCancel()
	snapshotReader, err := b.etcdClient.Snapshot(snapshotCTX)
	if err != nil {
		return fmt.Errorf("error trying to snapshot etcd: %w", err)
	}
	defer snapshotReader.Close()
	// write snapshot to buffer, tar header needs a size
	snapshotBytesBuff := &bytes.Buffer{}
	if _, err := io.Copy(snapshotBytesBuff, snapshotReader); err != nil {
		return fmt.Errorf("error copying etcd snapshot data to a buffer: %w", err)
	}

	// create backup buff, gzip and tar writers
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// write etcd snapshot to tar
	snapshotHdr := &tar.Header{
		Name:    "snapshot.db",
		Mode:    0600,
		Size:    int64(len(snapshotBytesBuff.Bytes())),
		ModTime: time.Now(),
	}
	err = tarWriter.WriteHeader(snapshotHdr)
	if err != nil {
		return fmt.Errorf("error writing etcd snapshot header to tar: %w", err)
	}
	if _, err = io.Copy(tarWriter, snapshotBytesBuff); err != nil {
		return fmt.Errorf("error writing etcd snapshot data to tar: %w", err)
	}

	// backup pki
	for _, pkiFile := range pkiFiles {
		pkiFilePath := path.Join(b.kubeadmPKIDirectory, pkiFile)
		f, err := os.Open(pkiFilePath)
		if err != nil {
			return fmt.Errorf("error opening pki file %s: %w", pkiFilePath, err)
		}

		stat, err := f.Stat()
		if err != nil {
			return fmt.Errorf("error stat pki file %s: %w", pkiFilePath, err)
		}

		pkiFileHeader := &tar.Header{
			Name:    path.Join("certs", pkiFile),
			Size:    stat.Size(),
			Mode:    int64(stat.Mode()),
			ModTime: stat.ModTime(),
		}

		err = tarWriter.WriteHeader(pkiFileHeader)
		if err != nil {
			return fmt.Errorf("error writing pki file %s header to tar %w", pkiFile, err)
		}

		if _, err = io.Copy(tarWriter, f); err != nil {
			return fmt.Errorf("error writing pki file %s to tar: %w", pkiFile, err)
		}
	}

	// close everything
	err = tarWriter.Close()
	if err != nil {
		return fmt.Errorf("error closing tar: %w", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		return fmt.Errorf("error closing gzip: %w", err)
	}

	// create backup
	now := time.Now()
	objectName := fmt.Sprintf("backup-%v.tar.gz", now.Format(time.RFC3339Nano))

	blobCreateCTX, blobCreateCTXCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer blobCreateCTXCancel()
	return b.blobClient.Create(blobCreateCTX, objectName, &buf)
}
