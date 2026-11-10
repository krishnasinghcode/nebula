// Package catalogue holds the read-only curated image catalogue.
// Image ids are Orbit-level names; DockerImage is the actual runtime image.
package catalogue

import "fmt"

// Image is a curated bootable template.
type Image struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DockerImage  string `json:"-"` // internal mapping, not part of the API
	OS           string `json:"os"`
	Description  string `json:"description"`
	SizeEstimate string `json:"size_estimate"`
}

// Images is the MVP catalogue. Built locally via scripts/build-images.sh.
var Images = []Image{
	{ID: "ubuntu-24.04", Name: "Ubuntu 24.04 LTS", DockerImage: "nebula/orbit-ubuntu-24.04:latest", OS: "ubuntu", Description: "Ubuntu 24.04 with SSH server pre-configured", SizeEstimate: "~200 MB"},
	{ID: "debian-12", Name: "Debian 12", DockerImage: "nebula/orbit-debian-12:latest", OS: "debian", Description: "Debian 12 with SSH server pre-configured", SizeEstimate: "~150 MB"},
}

// Get returns an image by id or nil.
func Get(id string) *Image {
	for i := range Images {
		if Images[i].ID == id {
			return &Images[i]
		}
	}
	return nil
}

// MustDockerImage maps an Orbit image id to its runtime image.
func MustDockerImage(id string) (string, error) {
	img := Get(id)
	if img == nil {
		return "", fmt.Errorf("unknown image %q", id)
	}
	return img.DockerImage, nil
}
