// environ provides information about the environment. This is mainly used for probing information about ECS
package environ

import (
	"errors"
	"net/url"
	"os"
)

// MetadataEnvVar is the environment variable made available by ECS within the container that holds the value
// of the URI to retrieve the container metadata
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html
const MetadataEnvVar = "ECS_CONTAINER_METADATA_URI_V4"

// IsECS returns true if the running environment is in ECS
func IsECS() bool {
	metadataUri := os.Getenv(MetadataEnvVar)

	if metadataUri == "" {
		return false
	}

	return true
}

// GetECSMetadataURI gets the URI from the ECS_CONTAINER_METADATA_URI_V4 env var
func GetECSMetadataURI() (*url.URL, error) {
	if !IsECS() {
		return nil, errors.New("Not running on ECS")
	}

	metadataUri := os.Getenv(MetadataEnvVar)
	return url.Parse(metadataUri)
}
