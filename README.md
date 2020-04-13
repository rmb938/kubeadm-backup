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

TODO

## TODO

* Support other blob storage backends
    - [X] GCS
    - [X] S3
    - [ ] Azure
    - [ ] Other blob storage?
- [X] Delete old backups
