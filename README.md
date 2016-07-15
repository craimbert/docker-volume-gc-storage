# Docker Volume Plugin: Google Cloud Storage
Reserving up front a persistent disk of several 10s of GB might not seem like the best idea for some simple container use cases, let's look at an on-demand elastic bucket storage platform: Google Cloud Storage.

## Overview
The goal is to use a storage bucket as a docker volume:
* a volume would be accessible to containers from completely different hosts, maybe not even running in the same cloud provider -> even local!
* direct access to the volume from the Google Cloud Storage web platform

## Screencast - Youtube video
[![screencast_picture](screenshots/screencast_picture.png?raw=true)](https://www.youtube.com/watch?v=4Z4mQqQ92tc)

## File System
Any volume needs to expose a file system to handle the "disk" I/O, however a storage bucket does not expose any.
<br/>
Solution: Google Cloud Platform `gcsfuse`:<br/>
`Cloud Storage FUSE is an open source Fuse adapter that allows you to mount Google Cloud Storage buckets as file systems on Linux or OS X systems`
* https://cloud.google.com/storage/docs/gcs-fuse
* https://github.com/GoogleCloudPlatform/gcsfuse

### Google Cloud Storage
Similar to Amazon S3, according to Google: `"Cloud Storage is typically used to store unstructured data. You can add objects of any kind and size, and up to 5 TB."` -> https://console.cloud.google.com/storage

## Installation

### Install Google Cloud Platform gcsfuse
https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/installing.md

### Install the Volume Driver
````
$ go get github.com/craimbert/docker-volume-gc-storage
$ go install github.com/craimbert/docker-volume-gc-storage
````

### Stop Docker engine (Debian)
````
$ service docker stop
````

### Generate on GCP a Service Account key in JSON format
Section `To generate a private key in JSON or PKCS12 format`:<br/> https://cloud.google.com/storage/docs/authentication?hl=en#generating-a-private-key

### Start the Volume Driver (Debian)
````
$ docker-volume-gc-storage -gcp-key-json gcp-srv-account-key.json
````

### Start Docker engine
````
$ service docker start
````

## Usage
- Create a volume
````
$ docker volume create --driver gcstorage --name datastore
datastore
````
The GCS Bucket name is defined by: **gcsProjectID_volumeName**
<br/><br/>
![gcs-bucket-0](screenshots/gcs-bucket-0.png?raw=true)
<br/><br/>
- List volumes
````
$ docker volume ls
DRIVER              VOLUME NAME
gcstorage           datastore
````
- Mount the volume on a container
````
$ docker run -it --rm -v datastore:/tmp alpine sh
/ # date > /tmp/date
````
The container dir `/tmp` is mounted on the GC Storage bucket defined by the volume `datastore` (through the host mountpoint `/var/lib/docker-volumes/gcstorage/datastore/_data`), the filesystem IO between host mountpoint and GCS bucket being handled by gcsfuse
<br/><br/>
![gcs-bucket-1](screenshots/gcs-bucket-1.png?raw=true)
<br/><br/>
Upload manually through Cloud Storage web portal a new `foo` file into the bucket:
<br/><br/>
![gcs-bucket-2](screenshots/gcs-bucket-2.png?raw=true)
<br/><br/>
Inside the container, the new file `foo` appears:
````
/tmp # ls -l /tmp/
total 1
-rw-r--r--    1 root     root            29 Jul  7 02:17 date
-rw-r--r--    1 root     root             4 Jul  7 03:50 foo
````
## Useful Links

### Google GO API
* https://godoc.org/google.golang.org/api/storage/v1
* https://godoc.org/google.golang.org/cloud/storage

### Docker Volume Driver interface
https://github.com/docker/go-plugins-helpers/blob/master/volume

##Tested environment: Go & Docker versions
```
$ go version
go version go1.6.2 linux/amd64

$ docker version
Client:
 Version:      1.11.2
 API version:  1.23
 Go version:   go1.5.4
 Git commit:   b9f10c9
 Built:        Wed Jun  1 21:23:39 2016
 OS/Arch:      linux/amd64
Server:
 Version:      1.11.2
 API version:  1.23
 Go version:   go1.5.4
 Git commit:   b9f10c9
 Built:        Wed Jun  1 21:23:39 2016
 OS/Arch:      linux/amd64
````
## Common Issues
!! NTP !!

## TODO
On Docker for Mac: Docker engine doesn't seem to register the Volume Driver (even after engine restart):
````
$ docker volume create --driver gcstorage --name datastore
Error response from daemon: create datastore: create datastore: Error looking up volume plugin gcstorage: plugin not found
```
When actually the plugin seems to be referenced properly and behaves correctly (checking via HTTP Remote API):
```
$ cat /etc/docker/plugins/gcstorage.spec
tcp://127.0.0.1:8080

$ curl \
    -X POST \
    -d '{}' \
    http://127.0.0.1:8080/Plugin.Activate
{"Implements": ["VolumeDriver"]}

$ curl \
    -X POST \
    -d '{}' \
    http://127.0.0.1:8080/VolumeDriver.List

$ curl \
    -X POST \
    -d '{"Name": "datastore", "Opts": {}}' \
    http://127.0.0.1:8080/VolumeDriver.Create

$ curl \
    -X POST \
    -d '{}' \
    http://127.0.0.1:8080/VolumeDriver.List
{"Mountpoint":"","Err":"","Volumes":[{"Name":"datastore","Mountpoint":"/var/lib/docker-volumes/gcstorage/datastore/_data"}],"Volume":null,"Capabilities":{"Scope":""}}

$ curl \
    -X POST \
    -d '{"Name": "datastore"}' \
    http://127.0.0.1:8080/VolumeDriver.Get
{"Mountpoint":"","Err":"","Volumes":null,"Volume":{"Name":"datastore","Mountpoint":"/var/lib/docker-volumes/gcstorage/datastore/_data"},"Capabilities":{"Scope":""}}

$ tree /var/lib/docker-volumes/gcstorage
/var/lib/docker-volumes/gcstorage
└── datastore
    └── _data
````
