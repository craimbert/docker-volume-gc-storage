package main

import (
	"fmt"
	"log"

	"github.com/docker/go-plugins-helpers/volume"
	gstorage "google.golang.org/api/storage/v1"
)

type gcpVolDriver struct {
	gcpClient         *gstorage.BucketsService
	gcpServiceKeyPath string
	gcpProjectID      string
	driverRootDir     string
	mountedBuckets    map[string]*gcsVolumes
}

type gcsVolumes struct {
	volume        *volume.Volume
	gcsBucketName string
	cleanCloud    bool
}

func newGcpVolDriver(driverRootDir, gcpServiceKeyPath string) (*gcpVolDriver, error) {
	log.Printf("GCP Volume Driver creation - Driver root dir: %s\n", driverRootDir)
	log.Printf("GCP Volume Driver creation - GCP Service Account key JSON: %s\n", gcpServiceKeyPath)
	gcpClient, err := newGoogleStorageBucketsService(gcpServiceKeyPath)
	if err != nil {
		return nil, err
	}
	gcpProjectID, err := getGCPProjectID(gcpServiceKeyPath)
	if err != nil {
		return nil, err
	}
	d := &gcpVolDriver{
		gcpClient:         gcpClient,
		gcpServiceKeyPath: gcpServiceKeyPath,
		gcpProjectID:      gcpProjectID,
		driverRootDir:     driverRootDir,
		mountedBuckets:    make(map[string]*gcsVolumes),
	}
	if err := d.syncWithHost(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *gcpVolDriver) Create(r volume.Request) volume.Response {
	log.Printf("Creation of volume '%s'...\n", r.Name)
	// Create a host mountpoint
	m, err := d.handleCreateMountpoint(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	// Create a bucket on GCP Storage
	bucketName, err := d.handleCreateGCStorageBucket(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	// Refer volumeName <-> gcsVolumes
	cleanCloud := false
	val, ok := r.Options["clean_cloud_bucket"]
	if ok && val == "yes" {
		cleanCloud = true
	}
	d.mountedBuckets[r.Name] = &gcsVolumes{
		volume: &volume.Volume{
			Name:       r.Name,
			Mountpoint: m,
		},
		gcsBucketName: bucketName,
		cleanCloud:    cleanCloud,
	}
	return volume.Response{}
}

func (d *gcpVolDriver) Remove(r volume.Request) volume.Response {
	log.Printf("Remove volume '%s'\n", r.Name)
	// Delete host mountpoint if necessary
	err := d.handleDeleteMountpoint(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	// Empty & Delete Google Cloud Storage bucket if necessary
	if err := d.handleRemoveGCStorageBucket(r.Name); err != nil {
		return volume.Response{Err: err.Error()}
	}
	// Remove the volume from the internal map
	delete(d.mountedBuckets, r.Name)
	return volume.Response{}
}

func (d *gcpVolDriver) Path(r volume.Request) volume.Response {
	return volume.Response{
		Mountpoint: d.getMountpoint(r.Name),
	}
}

func (d *gcpVolDriver) List(r volume.Request) volume.Response {
	var volumes []*volume.Volume
	for _, v := range d.mountedBuckets {
		volumes = append(volumes, v.volume)
	}
	return volume.Response{Volumes: volumes}
}

func (d *gcpVolDriver) Get(r volume.Request) volume.Response {
	mountedBucked, ok := d.mountedBuckets[r.Name]
	if ok {
		return volume.Response{
			Volume: mountedBucked.volume,
		}
	}
	return volume.Response{}
}

func (d *gcpVolDriver) Mount(r volume.Request) volume.Response {
	log.Printf("Mount volume '%s'\n", r.Name)
	// get mountpoint
	m := d.getMountpoint(r.Name)
	// mountpoint exists?
	exist, err := d.isPathExist(m)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	if !exist {
		return volume.Response{Err: fmt.Sprintf("Host mountpoint %s does not exist", m)}
	}
	// mount a GC Storage bucket using gcsfuse on the host mountpoint
	if err := d.mountGcsfuse(r.Name); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{
		Mountpoint: d.getMountpoint(r.Name),
	}
}

func (d *gcpVolDriver) Unmount(r volume.Request) volume.Response {
	log.Printf("Unmount volume '%s'\n", r.Name)
	// get mountpoint
	m := d.getMountpoint(r.Name)
	// mountpoint exists?
	exist, err := d.isPathExist(m)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	if !exist {
		return volume.Response{Err: fmt.Sprintf("Host mountpoint %s does not exist", m)}
	}
	// unmount the GC Storage bucket
	if err := d.unmountGcsfuse(r.Name); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (d *gcpVolDriver) Capabilities(r volume.Request) volume.Response {
	return volume.Response{Capabilities: volume.Capability{Scope: "global"}}
}
