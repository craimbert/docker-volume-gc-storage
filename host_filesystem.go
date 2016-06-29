package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/docker/go-plugins-helpers/volume"
)

// getVolumesFromHost looks up existing volumes defined in the volume driver root dir on the host
func (d *gcpVolDriver) getVolumesFromHost() ([]string, error) {
	var volumesNames []string
	// get all dirs from volume driver root dir
	existingVols, err := ioutil.ReadDir(d.driverRootDir)
	if err != nil {
		return nil, err
	}
	for _, v := range existingVols {
		if v.IsDir() {
			dataDir, err := ioutil.ReadDir(filepath.Join(d.driverRootDir, v.Name()))
			if err != nil {
				return nil, err
			}
			// a volume dir is defined by a path: volumeName/_data
			if len(dataDir) == 1 && dataDir[0].Name() == "_data" {
				volumesNames = append(volumesNames, v.Name())
			}
		}
	}
	return volumesNames, nil
}

// syncWithHost looks up potential existing volumes & creates GCStorage bucket if necessary
func (d *gcpVolDriver) syncWithHost() error {
	log.Println("Synchronizing: load existing volumes into driver & Google Cloud Storage")
	// get existing volumes defined for the driver
	volumesNames, err := d.getVolumesFromHost()
	if err != nil {
		return err
	}
	for _, v := range volumesNames {
		log.Printf("Synchronizing: existing volume '%s' found\n", v)
		// create a GCStorage bucket for that volume if not exist
		bucketName, err := d.handleCreateGCStorageBucket(v)
		if err != nil {
			return err
		}
		// add this volume to the driver's in-memory map of volumes
		d.mountedBuckets[v] = &gcsVolumes{
			volume: &volume.Volume{
				Name:       v,
				Mountpoint: filepath.Join(d.driverRootDir, v, "_data"),
			},
			gcsBucketName: bucketName,
			cleanCloud:    true,
		}
	}
	return nil
}

// getMountpoint defines a mountpoint path from driver root dir & volume name
func (d *gcpVolDriver) getMountpoint(name string) string {
	return filepath.Join(d.driverRootDir, name, "_data")
}

// createMountpoint creates a mountpoint dir from a path
func (d *gcpVolDriver) createMountpoint(mountpoint string) error {
	if err := os.MkdirAll(mountpoint, 0755); err != nil {
		return err
	}
	log.Printf("Mountpoint %s created on host\n", mountpoint)
	return nil
}

// isPathExist returns true if a path exists on the host, false otherwise
func (d *gcpVolDriver) isPathExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// handleCreateMountpoint creates a host mountpoint
func (d *gcpVolDriver) handleCreateMountpoint(volumeName string) (string, error) {
	// Create mountpoint dir on local host
	m := d.getMountpoint(volumeName)
	// mountpoint already exists?
	exist, err := d.isPathExist(m)
	if err != nil {
		return "", err
	}
	if exist {
		return "", fmt.Errorf("Host mountpoint %s already exist", m)
	}
	if err := d.createMountpoint(m); err != nil {
		return "", err
	}
	return m, nil
}

// deleteMountpoint deletes the mountpoint directory
func (d *gcpVolDriver) deleteMountpoint(mountpoint string) error {
	if err := os.RemoveAll(mountpoint); err != nil {
		return err
	}
	log.Printf("Mountpoint %s deleted on host\n", mountpoint)
	return nil
}

// handleDeleteMountpoint deletes a host mountpoint
func (d *gcpVolDriver) handleDeleteMountpoint(volumeName string) error {
	m := d.getMountpoint(volumeName)
	// mountpoint already exists?
	exist, err := d.isPathExist(m)
	if err != nil {
		return err
	}
	if exist {
		// delete mountpoint
		if err := d.deleteMountpoint(filepath.Dir(m)); err != nil {
			return err
		}
	}
	return nil
}
