package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/rmb938/kubeadm-backup/pkg/backup"
	"github.com/rmb938/kubeadm-backup/pkg/blob"
	"github.com/rmb938/kubeadm-backup/pkg/etcd"
)

func main() {
	logLevel := flag.Int("v", 0, "number for the log level verbosity")

	// etcd flags
	etcdEndpoint := flag.String("etcd-endpoint", "http://127.0.0.1:2379", "etcd endpoint to connect to")
	etcdCaFile := flag.String("etcd-ca-file", "", "etcd ca to use")
	etcdKeyFile := flag.String("etcd-key-file", "", "etcd key to use")
	etcdCertFile := flag.String("etcd-certificate-file", "", "etcd certificate to use")

	// kubeadm flags
	kubeadmPKIDirectory := flag.String("kubeadm-pki-directory", "", "the directory for kubeadm pki")

	// backup flags
	backupDuration := flag.Duration("backup-interval", 1*time.Hour, "how often to take a backup")
	backupTTL := flag.Duration("backup-ttl", (30*24)*time.Hour, "backup retention period")

	// blob flags
	blobConfigFile := flag.String("blob-config-file", "", "Path to blob storage configuration file")

	flag.Parse()

	zapConfig := zap.NewProductionConfig()
	zapConfig.DisableStacktrace = true
	zapConfig.DisableCaller = true
	zapConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(0 - *logLevel))

	zapLog, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("error creating logger: %v", err))
	}
	defer zapLog.Sync()

	logr := zapr.NewLogger(zapLog)
	setupLog := logr.WithName("setup")

	if *blobConfigFile == "" {
		setupLog.Error(fmt.Errorf("blob-config-file not set"), "invalid command flags")
		os.Exit(1)
	}

	if *kubeadmPKIDirectory == "" {
		setupLog.Error(fmt.Errorf("kubeadm-pki-directory not set"), "invalid command flags")
		os.Exit(1)
	}

	if (*etcdKeyFile != "" && *etcdCertFile == "") || (*etcdKeyFile == "" && *etcdCertFile != "") {
		setupLog.Error(fmt.Errorf("both etcd-key-file and etcd-certificate-file must be given"), "invalid command flags")
		os.Exit(1)
	}

	setupLog.Info("Creating Blob Client")
	blobClient, err := blob.CreateBlobClientFromConfig(*blobConfigFile)
	if err != nil {
		setupLog.Error(err, "error creating blob client from config")
		os.Exit(1)
	}
	defer blobClient.Close()

	setupLog.Info("Creating etcd Client")
	etcdClient, err := etcd.NewEtcdClient(*etcdEndpoint, *etcdCaFile, *etcdKeyFile, *etcdCertFile)
	if err != nil {
		setupLog.Error(err, "Error creating etcd client")
		os.Exit(1)
	}
	defer etcdClient.Close()

	backupTimer := backup.NewBackupTimer(blobClient, etcdClient, *kubeadmPKIDirectory, *backupDuration, *backupTTL, logr.WithName("backup-timer"))
	backupTimer.Run()

}
