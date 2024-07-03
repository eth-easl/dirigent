package image_storage

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type OCIImageStorage struct {
	DefaultImageStorage

	httpClient *http.Client
}

type ociManifest struct {
	Layers []struct {
		Sum string `json:"blobSum"`
	} `json:"fsLayers"`
}

func NewOCIImageStorage() *OCIImageStorage {
	return &OCIImageStorage{
		*NewDefaultImageStorage(),
		&http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				IdleConnTimeout:     5 * time.Second,
				MaxIdleConns:        25,
				MaxIdleConnsPerHost: 25,
			},
		},
	}
}

func (s *OCIImageStorage) fetchImageSize(image string) (uint64, error) {
	logrus.Tracef("Fetching image size for %s", image)
	imageHost, imageRepoAndTag, _ := strings.Cut(image, "/")
	imageRepo, imageTag, _ := strings.Cut(imageRepoAndTag, ":")
	manifestUrl := fmt.Sprintf("http://%s/v2/%s/manifests/%s", imageHost, imageRepo, imageTag)
	manifestResp, err := s.httpClient.Get(manifestUrl)
	if err != nil {
		return 0, err
	}

	defer manifestResp.Body.Close()
	manifestBody, err := io.ReadAll(manifestResp.Body)
	if err != nil {
		return 0, err
	}

	manifest := ociManifest{}
	err = json.Unmarshal(manifestBody, &manifest)
	if err != nil {
		return 0, err
	}

	imageSize := uint64(0)
	for _, layerReference := range manifest.Layers {
		layerUrl := fmt.Sprintf("http://%s/v2/%s/blobs/%s", imageHost, imageRepo, layerReference.Sum)
		layerResp, err := s.httpClient.Head(layerUrl)
		if err != nil {
			return 0, err
		}
		imageSize += uint64(layerResp.ContentLength)
	}
	return imageSize, nil
}

func (s *OCIImageStorage) RegisterWithFetch(image string, node core.WorkerNodeInterface) error {
	if node.AddImage(image) {
		size, err := s.fetchImageSize(image)
		if err != nil {
			return err
		}
		s.registerImage(image, size)
		logrus.Tracef("Registered image %s of size %d for node %s", image, size, node.GetName())
	}
	return nil
}
