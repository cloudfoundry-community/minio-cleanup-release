# Minio Cleanup Release

A [BOSH](https://bosh.io) release designed to periodically clean up a [Minio](https://minio.io) server.

## Getting Started

This release is designed to be colocated on a BOSH-managed Minio server. It works directly against the backing
filesystem, and as such, is not designed to work against a remote S3 store.

```sh
$ git clone https://github.com/pivotal/minio-cleanup-release
$ cd minio-cleanup-release
$ bosh upload-release
```

## Example 

```yaml
name: minio

releases:
- name: minio
  version: latest
- name: minio-cleanup
  version: latest

stemcells:
- alias: default
  os: ubuntu-xenial
  version: latest

variables:
- name: minio_accesskey
  type: password
- name: minio_secretkey
  type: password

instance_groups:
- name: minio
  azs: [z1]
  instances: 1
  jobs:
  - name: minio-server
    release: minio
    templates:
    - name: minio-server
    provides: 
      minio-server: {as: minio-link}
    properties:
      credential:
        accesskey: ((minio_accesskey))
        secretkey: ((minio_secretkey))
      port: 9000
  - name: minio-cleaner
    release: minio-cleanup
    properties:
      base_directory: /var/vcap/store/minio-server
      schedule: "@weekly" # This is a cron-compliant schedule
      buckets:
        bucket1:
        - pattern: cf-(.*).pivotal
          retain: 3
        - pattern: ops-manager-(.*).ova
          retain: 5
  networks:
  - name: default
  vm_type: small
  persistent_disk_type: '10GB'
  stemcell: default
- name: tests
  azs: [z1]
  instances: 1
  lifecycle: errand
  post_deploy: true
  jobs:
  - name: smoke-tests
    release: minio
    templates:
    - name: smoke-tests
    consumes:
      minio: {from: minio-link}
  networks:
  - name: default
  vm_type: small
  stemcell: default
- name: bucket-seeding # To create default buckets after manifest-deploy
  azs: [z1]
  instances: 1
  lifecycle: errand
  post_deploy: true
  jobs:
  - name: mc
    release: minio
    templates:
    - name: mc
    consumes:
      minio: {from: minio-link}
    properties:
      script: |
        #!/bin/sh
        mc mb myminio/bucket1
        mc mb myminio/bucket2
        mc mb myminio/bucket3
        mc mb myminio/bucket4
  networks:
  - name: default
  vm_type: small
  stemcell: default

update:
  canaries: 1
  canary_watch_time: 1000-30000
  update_watch_time: 1000-30000
  max_in_flight: 1
```