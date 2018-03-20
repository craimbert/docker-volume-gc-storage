package main

import (
	"fmt"
	"log"
	"errors"

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

func (d *gcpVolDriver) Create(r *volume.CreateRequest) error {
	log.Printf("Creation of volume '%s'...\n", r.Name)
	// Create a host mountpoint
	m, err := d.handleCreateMountpoint(r.Name)
	if err != nil {
		return err
	}
	// Create a bucket on GCP Storage
	bucketName, err := d.handleCreateGCStorageBucket(r.Name)
	if err != nil {
		return err
	}
	// Refer volumeName <-> gcsVolumes
	cleanCloud := true
	val, ok := r.Options["clean_cloud_bucket"]
	if ok && val == "no" {
		cleanCloud = false
	}
	d.mountedBuckets[r.Name] = &gcsVolumes{
		volume: &volume.Volume{
			Name:       r.Name,
			Mountpoint: m,
		},
		gcsBucketName: bucketName,
		cleanCloud:    cleanCloud,
	}
	return nil
}

func (d *gcpVolDriver) Remove(r *volume.RemoveRequest) error {
	log.Printf("Remove volume '%s'\n", r.Name)
	// Delete host mountpoint if necessary
	err := d.handleDeleteMountpoint(r.Name)
	if err != nil {
		return err
	}
	// Empty & Delete Google Cloud Storage bucket if necessary
	if err := d.handleRemoveGCStorageBucket(r.Name); err != nil {
		return err
	}
	// Remove the volume from the internal map
	delete(d.mountedBuckets, r.Name)
	return nil
}

func (d *gcpVolDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	return &volume.PathResponse{
		Mountpoint: d.getMountpoint(r.Name),
	}, nil
}

func (d *gcpVolDriver) List() (*volume.ListResponse, error) {
	var volumes []*volume.Volume
	for _, v := range d.mountedBuckets {
		volumes = append(volumes, v.volume)
	}
	return &volume.ListResponse{Volumes: volumes}, nil
}

func (d *gcpVolDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	mountedBucked, ok := d.mountedBuckets[r.Name]
	if ok {
		return &volume.GetResponse{
			Volume: mountedBucked.volume,
		}, nil
	}
	return nil, errors.New("Failed to get mountted bucket")
}

func (d *gcpVolDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	log.Printf("Mount volume '%s'\n", r.Name)
	// get mountpoint
	m := d.getMountpoint(r.Name)
	// mountpoint exists?
	exist, err := d.isPathExist(m)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, errors.New(fmt.Sprintf("Host mountpoint %s does not exist", m))
	}
	// mount a GC Storage bucket using gcsfuse on the host mountpoint
	if err := d.mountGcsfuse(r.Name); err != nil {
		return nil, err
	}
	return &volume.MountResponse{
		Mountpoint: d.getMountpoint(r.Name),
	}, nil
}

func (d *gcpVolDriver) Unmount(r *volume.UnmountRequest) error {
	log.Printf("Unmount volume '%s'\n", r.Name)
	// get mountpoint
	m := d.getMountpoint(r.Name)
	// mountpoint exists?
	exist, err := d.isPathExist(m)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New(fmt.Sprintf("Host mountpoint %s does not exist", m))
	}
	// unmount the GC Storage bucket
	if err := d.unmountGcsfuse(r.Name); err != nil {
		return err
	}
	return nil
}

func (d *gcpVolDriver) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "global"}}
}
