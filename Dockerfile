ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest

ARG ARCH="amd64"
ARG OS="linux"
COPY bin/kubeadm-backup-${OS}-${ARCH} /bin/kubeadm-backup

ENTRYPOINT [ "/bin/kubeadm-backup" ]
