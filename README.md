# Kubeadm Backup
A tool to backup and restore kubeadm clusters

## Why another backup and restore tool?

Not every backup tool is perfect. Depending on the how the cluster was made, what workloads are running in the cluster,
and the failure scenario different tools may be required.

Kubeadm Backup focuses on the scenario where only the Kubernetes masters were lost. In this case application uptime
may be the priority. This means that workers must stay online, existing credentials like kubelet certificates and
service account keys must continue to be valid on recovery, and most importantly there must be minimal impact to applications
that are still healthy and running.

Kubeadm Backup aims to fulfill those requirements by using a different approach. By taking etcd snapshots and backing 
up Kubeadm CA certificates existing credentials will continue to be valid, workers will stay online and be able to 
automatically rejoin the Kubernetes cluster, and applications will stay running on the nodes they were scheduled on. 

## Usage

### Requirements
  * A Kubernetes Cluster built using kubeadm
  * A blob storage bucket with credentials
  
### Deployment

An example deployment can be found by running `kustomize build kustomize/kubeadm`

### Command Line Flags

```shell script
  -backup-interval duration
        how often to take a backup (default 1h0m0s)
  -backup-ttl duration
        backup retention period (default 720h0m0s)
  -blob-config-file string
        Path to blob storage configuration file
  -etcd-ca-file string
        etcd ca to use
  -etcd-certificate-file string
        etcd certificate to use
  -etcd-endpoint string
        etcd endpoint to connect to (default "http://127.0.0.1:2379")
  -etcd-key-file string
        etcd key to use
  -kubeadm-pki-directory string
        the directory for kubeadm pki
  -v int
        number for the log level verbosity
```
  
### Configuration

#### GCS

To configure Google Cloud Storage bucket as an blob store you need to set the bucket with GCS bucket name and configure Google Application credentials.

For example:

```yaml
type: GCS
config:
  bucket: ""
  service_account: ""
```

##### Using GOOGLE_APPLICATION_CREDENTIALS

Application credentials are configured via a JSON file and only the bucket needs to be specified, the client looks for:

1. A JSON file whose path is specified by the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.
1. A JSON file in a location known to the gcloud command-line tool. On Windows, this is `%APPDATA%/gcloud/application_default_credentials.json.` On other systems, `$HOME/.config/gcloud/application_default_credentials.json`.
1. On Google App Engine it uses the `appengine.AccessToken` function.
1. On Google Compute Engine and Google App Engine Managed VMs, it fetches credentials from the metadata server.

You can read more on how to get application credential json file in https://cloud.google.com/docs/authentication/production

##### Using an inline Service Account

Another possibility is to inline the GCP service account into the configuration.

```yaml
type: GCS
config:
    bucket: ""
    service_account: |-
      {
            "type": "service_account",
            "project_id": "project",
            "private_key_id": "abcdefghijklmnopqrstuvwxyz12345678906666",
            "private_key": "-----BEGIN PRIVATE KEY-----\...\n-----END PRIVATE KEY-----\n",
            "client_email": "project@kubeadmbackup.iam.gserviceaccount.com",
            "client_id": "123456789012345678901",
            "auth_uri": "https://accounts.google.com/o/oauth2/auth",
            "token_uri": "https://oauth2.googleapis.com/token",
            "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
            "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/kubeadmbackup%40gitpods.iam.gserviceaccount.com"
          }
```

#### S3

Kubeadm Backup uses the [minio client](https://github.com/minio/minio-go) library to backups to S3.

```yaml
type: S3
config:
    endpoint: ""
    insecure: false
    bucket: ""
    access_key: ""
    secret_key: ""
    http_config:
      idle_conn_timeout: 90s
      response_header_timeout: 2m
      insecure_skip_verify: false
```

At a minimum, you will need to set `bucket`, `endpoint`, `access_key`, and `secret_key` keys. The rest of the keys are optional.

The AWS region to endpoint mapping can be found in this [link](https://docs.aws.amazon.com/general/latest/gr/s3.html).

## TODO

* Support other blob storage backends
    - [X] GCS
    - [X] S3
    - [ ] Azure
    - [ ] Other blob storage?
- [X] Delete old backups
