package main

import (
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

func UpdateContainerConcurrency() {
	data := url.Values{}
	data.Set("function", "test")
	data.Set("container_concurrency", "1")
	resp, err := http.PostForm("http://127.0.0.1:9091/patch", data)
	if err != nil || resp.StatusCode != 200 {
		logrus.Errorf("Failed to update container concurrency")
	}
}

func main() {
	UpdateContainerConcurrency()
}
