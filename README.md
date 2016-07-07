# Google Cloud Storage
###Generate on GCP a Service Account key as JSON file
Section `generate a private key in JSON `: https://cloud.google.com/storage/docs/authentication?hl=en#generating-a-private-key


# Installation
##Install Google Cloud Platform `gcsfuse`
https://github.com/GoogleCloudPlatform/gcsfuse/blob/master/docs/installing.md
##Install the Volume Driver
````
$ go get github.com/craimbert/docker-volume-gc-storage
$ cd $GOPATH/src/github.com/craimbert/docker-volume-gc-storage && go get -d -v
$ go install craimbert/docker-volume-gc-storage
````
##Stop Docker engine
````
$ service docker stop
````
##Start the Volume Driver
````
$ docker-volume-gcp-storage -gcp-key-json gcp-srv-account-key.json
````
##Stop Docker engine
````
$ service docker start
````
#Using the Volume Driver
````
$ docker volume create --driver gcstorage --name datastore
datastore

$ docker volume ls
DRIVER              VOLUME NAME
gcstorage           datastore

$ docker run -it --rm -v datastore:/tmp alpine sh
/ # date > /tmp/date
````
# Useful Links
###Google Cloud Storage - Buckets
https://console.cloud.google.com/storage

###Google Cloud Storage - FUSE
* https://cloud.google.com/storage/docs/gcs-fuse
* https://github.com/GoogleCloudPlatform/gcsfuse


###Google GO API
* https://godoc.org/google.golang.org/api/storage/v1
* https://godoc.org/google.golang.org/cloud/storage

###Docker Volume Driver interface
https://github.com/docker/go-plugins-helpers/blob/master/volume

#Tested environment: Go & Docker versions
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
# Common Issues
!! NTP !!

#TODO
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
