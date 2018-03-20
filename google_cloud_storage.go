package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	gstorage "google.golang.org/api/storage/v1"
	//cloud "cloud.google.com/go"
	gcloudstorage "cloud.google.com/go/storage"
)

// getGCPProjectID returns the unique ID of the Google Cloud Platform project defined in the service key JSON
func getGCPProjectID(jsonKeyPath string) (string, error) {
	var gcpServiceAccount map[string]string
	file, err := ioutil.ReadFile(jsonKeyPath)
	if err != nil {
		return "", nil
	}
	json.Unmarshal(file, &gcpServiceAccount)
	gcpProjectID := gcpServiceAccount["project_id"]
	log.Printf("GCP Project ID: %s\n", gcpProjectID)
	return gcpProjectID, nil
}

// getGCPBucketName defines the name of a GCStorage bucket based on GCP project ID & volume name
func (d *gcpVolDriver) getGCPBucketName(volumeName string) string {
	return fmt.Sprintf("%s_%s", d.gcpProjectID, volumeName)
}

// newGoogleStorageBucketsService creates a GCStorage BucketService from the GCP service key file
func newGoogleStorageBucketsService(keyfilePath string) (*gstorage.BucketsService, error) {
	jsonKey, err := ioutil.ReadFile(keyfilePath)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(
		jsonKey,
		gstorage.CloudPlatformScope,
	)
	if err != nil {
		return nil, err
	}
	storageService, err := gstorage.New(conf.Client(oauth2.NoContext))
	if err != nil {
		return nil, err
	}
	return storageService.Buckets, err
}

// newGoogleCloudStorageClient creates a Google Cloud Platform client used for BucketService unsupported actions
func newGoogleCloudStorageClient(keyfilePath string) (*gcloudstorage.Client, error) {
	jsonKey, err := ioutil.ReadFile(keyfilePath)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(
		jsonKey,
		gcloudstorage.ScopeFullControl,
	)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := gcloudstorage.NewClient(
		ctx,
		//cloud.WithTokenSource(conf.TokenSource(ctx)),
	)
	return client, err
}

// emptyGCSBucket empties the content of a Google Cloud Storage bucket, without deleting the bucket itself
func (d *gcpVolDriver) emptyGCSBucket(client *gcloudstorage.Client, bucketName string) error {
	bucketHandler := client.Bucket(bucketName)
	ctx := context.Background()
	list := bucketHandler.Objects(ctx, &gcloudstorage.Query{})
	for {
		r, err := list.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		log.Printf("Deleting object '%+s' from GCS Bucket '%s'\n", r.Name, bucketName)
		if err := bucketHandler.Object(r.Name).Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}

// IsGCSBucketExist returns true if a GCStorage bucket with a name GCPprojectID_volumeName exists
func (d *gcpVolDriver) IsGCSBucketExist(bucketName string) (bool, error) {
	buckets, err := d.gcpClient.List(d.gcpProjectID).Do()
	if err != nil {
		return false, err
	}
	for _, b := range buckets.Items {
		if b.Name == bucketName {
			log.Printf("Google Cloude Storage bucket '%s' already exists\n", bucketName)
			return true, nil
		}
	}
	log.Printf("There is no bucket named '%s' on Google Cloud Storage\n", bucketName)
	return false, nil
}

// createGCPStorageBucket creates a bucket on GCStorage from its name
func (d *gcpVolDriver) createGCPStorageBucket(bucketName string) (*gstorage.Bucket, error) {
	bucket, err := d.gcpClient.Insert(
		d.gcpProjectID,
		&gstorage.Bucket{
			Name:         bucketName,
			Location:     "US",
			StorageClass: "STANDARD",
		}).Do()
	if err != nil {
		return nil, err
	}
	log.Printf("Google Cloud Storage Bucket '%s' created for the project '%s'\n", bucketName, d.gcpProjectID)
	return bucket, nil
}

// handleCreateGCStorageBucket handles the safe creation of a GCStorage from its name
func (d *gcpVolDriver) handleCreateGCStorageBucket(volumeName string) (string, error) {
	bucketName := d.getGCPBucketName(volumeName)
	bucketExist, err := d.IsGCSBucketExist(bucketName)
	if err != nil {
		return "", err
	}
	if !bucketExist {
		_, err := d.createGCPStorageBucket(bucketName)
		if err != nil {
			return "", err
		}
	}
	return bucketName, nil
}

// deleteStorageBucket deletes a bucket on GCStorage by its name
func (d *gcpVolDriver) deleteStorageBucket(bucketName string) error {
	if err := d.gcpClient.Delete(bucketName).Do(); err != nil {
		return err
	}
	log.Printf("Google Cloud Storage Bucket '%s' deleted\n", bucketName)
	return nil
}

// handleRemoveGCStorageBucket handles the safe deletion of a GCStorage by its name
func (d *gcpVolDriver) handleRemoveGCStorageBucket(volumeName string) error {
	bucketName := d.getGCPBucketName(volumeName)
	bucketExist, err := d.IsGCSBucketExist(bucketName)
	if err != nil {
		return err
	}
	if bucketExist && d.mountedBuckets[volumeName].cleanCloud {
		// Empty the bucket
		client, err := newGoogleCloudStorageClient(d.gcpServiceKeyPath)
		if err != nil {
			return err
		}
		if err := d.emptyGCSBucket(client, bucketName); err != nil {
			return err
		}
		// Delete the bucket on GCP Storage
		if err := d.deleteStorageBucket(bucketName); err != nil {
			return err
		}
	}
	return nil
}
