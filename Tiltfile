# -*- mode: Python -*-


def indent(text, amount, ch=' '):
    padding = amount * ch
    output = ''
    for line in text.splitlines(True):
        output += padding + line
    return output


# set defaults
settings = {
    "minio": True,
    "blob_bucket": "kubeadm-backup-dev",

    # S3 Configuration
    # "s3": {
    #     "endpoint": "", # https://docs.aws.amazon.com/general/latest/gr/s3.html
    #     "access_key": "",
    #     "secret_key": ""
    # },

    # GCS Configuration
    # "gcs": {
    #     "service_account_file": ""
    # }
}

# global settings
settings.update(read_json(
    "tilt-settings.json",
    default={},
))

blob_config = None

# if minio is set deploy minio use it's s3 creds
if settings['minio']:
    blob_config = '''
type: S3
config:
    endpoint: "minio:9000"
    insecure: true
    bucket: "kubeadm-backup-dev"
    access_key: "minio"
    secret_key: "minio123"
'''

    # deploy minio
    k8s_yaml(kustomize('kustomize/minio'))

    # port forward minio
    k8s_resource('minio', port_forwards="9000")
else:

    # s3 is given so lets use that
    if 's3' in settings:
        blob_config = '''
type: S3
config:
    endpoint: "%(endpoint)s"
    bucket: "%(blob_bucket)s"
    access_key: "%(access_key)s"
    secret_key: "%(secret_key)s"
''' % {'blob_bucket': settings['blob_bucket'], 'endpoint': '', 'access_key': '', 'secret_key': ''}

    # gcs is given so lets use that
    elif 'gcs' in settings:
        blob_config = '''
type: GCS
config:
    bucket: "%(blob_bucket)s"
    service_account: %(gcs_service_account)s
''' % {'blob_bucket': settings['blob_bucket'], 'gcs_service_account': ''}

    # no blob storage creds given
    else:
        fail('Valid blob credentials must be given when minio is false')

if blob_config == None:
    fail('blob config file is None for some reason')

blob_config = blob_config.strip()

blob_secret = '''
apiVersion: v1
kind: Secret
metadata:
    name: kubeadm-backup-blob-config
stringData:
    config.yaml: |-
%(blob_config)s
''' % {'blob_config': indent(blob_config, 8)}

local_resource(
    'kubeadm-backup-binary',
    cmd='make build-amd64',
    deps=[
        'cmd',
        'pkg'
    ],
)

custom_build(
    'kubeadm-backup',
    'docker build -t $EXPECTED_REF .',
    deps=[
        'bin/kubeadm-backup-linux-amd64',
        'Dockerfile'
    ],
)

# apply blob secret
k8s_yaml(blob(blob_secret))

k8s_resource('kubeadm-backup', resource_deps=['minio'])

# apply kustomize
k8s_yaml(kustomize('kustomize/tilt'))
