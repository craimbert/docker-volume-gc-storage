package main

import (
	"log"
	"os/exec"
)

// mountGcsfuse mounts a GCStorage bucket on a host dir using gcsfuse
func (d *gcpVolDriver) mountGcsfuse(volumeName string) error {
	// get host mountpoint path
	m := d.getMountpoint(volumeName)
	// get GCS bucket name
	bucketName := d.getGCPBucketName(volumeName)
	// mount GCStorage bucket on host mounpoint
	log.Printf("Mounting host mountpoint '%s' to Google Cloud Storage Bucket '%s'\n", m, bucketName)
	log.Printf("Running: $ gcsfuse --key-file %s %s %s\n", d.gcpServiceKeyPath, bucketName, m)
	cmd := exec.Command("gcsfuse", "--key-file", d.gcpServiceKeyPath, bucketName, m)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// unmountGcsfuse unmounts a mounted GCStorage bucket on a host dir
func (d *gcpVolDriver) unmountGcsfuse(volumeName string) error {
	// get host mountpoint path
	m := d.getMountpoint(volumeName)
	// get GCS bucket name
	bucketName := d.getGCPBucketName(volumeName)
	// unmount the GCS bucket
	log.Printf("Unmounting host mountpoint '%s' from Google Cloud Storage Bucket '%s'\n", m, bucketName)
	log.Printf("Running: $ fusermount -u %s\n", m)
	cmd := exec.Command("fusermount", "-u", m)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
