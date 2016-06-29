#Google Cloud Storage - FUSE
https://cloud.google.com/storage/docs/gcs-fuse
https://github.com/GoogleCloudPlatform/gcsfuse

## Generate on GCP a Service Account key as JSON file
https://cloud.google.com/storage/docs/authentication?hl=en#generating-a-private-key


# Docker Volume Plugin Interface
https://github.com/docker/go-plugins-helpers/blob/master/volume/api.go

# Google GO API
https://godoc.org/google.golang.org/api/storage/v1
https://godoc.org/google.golang.org/cloud/storage



!! NTP !!

// blocked by running serveTCP on Docker for Mac native:
````
$ docker volume create --driver gcstorage --name datastore
Error response from daemon: create datastore: create datastore: Error looking up volume plugin gcstorage: plugin not found

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
