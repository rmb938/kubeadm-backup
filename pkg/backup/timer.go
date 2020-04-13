package backup

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/go-logr/logr"

	"github.com/rmb938/kubeadm-backup/pkg/blob"
	"github.com/rmb938/kubeadm-backup/pkg/etcd"
)

type backupTimer struct {
	blobClient blob.BlobClient
	etcdClient *etcd.Client

	kubeadmPKIDirectory string

	interval time.Duration
	ttl      time.Duration

	log logr.Logger
}

func NewBackupTimer(blobClient blob.BlobClient, etcdClient *etcd.Client, kubeadmPKIDirectory string, interval time.Duration, ttl time.Duration, log logr.Logger) *backupTimer {
	return &backupTimer{
		blobClient: blobClient,
		etcdClient: etcdClient,

		kubeadmPKIDirectory: kubeadmPKIDirectory,

		interval: interval,
		ttl:      ttl,

		log: log,
	}
}

func (bt *backupTimer) Run() {
	ticker := time.NewTicker(bt.interval)
	defer ticker.Stop()

	if err := bt.cleanBackups(); err != nil {
		bt.log.Error(err, "error cleaning backups")
		os.Exit(1) // failed to take the first backup so just exist
	}

	if err := bt.doBackup(); err != nil {
		bt.log.Error(err, "error taking backup")
		os.Exit(1) // failed to take the first backup so just exist
	}

	for {
		select {
		case <-ticker.C:
			if err := bt.cleanBackups(); err != nil {
				bt.log.Error(err, "error cleaning backups")
			}
			if err := bt.doBackup(); err != nil {
				bt.log.Error(err, "error taking backup")
			}
		}
	}
}

func (bt *backupTimer) doBackup() error {
	bt.log.Info("taking backup")
	b := backup{
		blobClient:          bt.blobClient,
		etcdClient:          bt.etcdClient,
		kubeadmPKIDirectory: bt.kubeadmPKIDirectory,
	}
	err := b.Take()
	if err != nil {
		return err
	}
	bt.log.Info("backup done")
	return nil
}

func (bt *backupTimer) cleanBackups() error {
	bt.log.Info("cleaning old backups")

	listCTX, listCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer listCancel()
	objectNamesChan := bt.blobClient.List(listCTX)

	for objInterface := range objectNamesChan {
		switch objInterface.(type) {
		case error:
			return objInterface.(error)
		case string:
			objectName := objInterface.(string)

			objectTime, err := time.Parse(time.RFC3339Nano, objectName[7:len(objectName)-7])
			if err != nil {
				return errors.Wrapf(err, "error parsing backup time for object %s", objectName)
			}

			now := time.Now()

			if now.After(objectTime.Add(bt.ttl)) {
				bt.log.Info("Deleting old backup", "backup-time", objectTime.Format(time.RFC3339Nano))

				deleteCTX, deleteCancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer deleteCancel()
				err = bt.blobClient.Delete(deleteCTX, objectName)
				if err != nil {
					return errors.Wrapf(err, "error deleting old backup taken at %v", objectTime.Format(time.RFC3339Nano))
				}

				bt.log.Info("Deleted old backup", "backup-time", objectTime.Format(time.RFC3339Nano))
			}
		default:
			return errors.Errorf("Unknown type from objects channel: %T", objInterface)
		}
	}

	bt.log.Info("done cleaning old backups")
	return nil
}
