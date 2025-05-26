package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

const (
	findReposFmt = `https://quay.io/api/v1/find/repositories?page=%d&includeUsage=false`
	listTagsFmt  = `https://quay.io/api/v1/repository/%s/%s/tag`
)

type RepoResponse struct {
	Results []*Repository `json:"results"`
}

type Repository struct {
	Namespace *Namespace `json:"namespace"`
	Name      string     `json:"name"`
}

type Namespace struct {
	Name string `json:"name"`
}

type TagResponse struct {
	Tags []*Tag `json:"tags"`
}

type Tag struct {
	Name string `json:"name"`
}

type ImageMetadata struct {
	Name      string `json:"name"`
	NumLayers int    `json:"num_layers"`
	TotalSize int64  `json:"total_size_bytes"`
}
func fetchImages() []string {
	fetchRepos := func(page int) []*Repository {
		fmt.Printf("fetching page: %d\n", page)
		resp, err := http.Get(fmt.Sprintf(findReposFmt, page))
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		var repoResp RepoResponse
		if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
			panic(err)
		}

		return repoResp.Results
	}

	fetchTags := func(repo *Repository) []string {
		resp, err := http.Get(fmt.Sprintf(listTagsFmt, repo.Namespace.Name, repo.Name))
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		var tagsResp TagResponse
		if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
			panic(err)
		}

		tags := make([]string, 0, 10)
		for i := 0; i < len(tagsResp.Tags) && i < cap(tags); i++ {
			tag := tagsResp.Tags[i].Name
			if strings.HasSuffix(tag, ".sig") || strings.HasSuffix(tag, ".sbom") {
				continue
			}
			tags = append(tags, tag)
		}

		return tags
	}

	images := make([]string, 0, 2500)
	all_refs := map[string]any{}
	for i := range 30 {
		repos := fetchRepos(i)
		for _, repo := range repos {
			tags := fetchTags(repo)
			for _, tag := range tags {
				// Ignore AI (they're too big): modh, notebook
				if strings.Contains(repo.Name, "skynet") {
					continue
				}
				ref := fmt.Sprintf("quay.io/%s/%s:%s", repo.Namespace.Name, repo.Name, tag)
				if _, ok := all_refs[ref]; ok {
					continue
				}
				fmt.Printf("new ref: %s\n", ref)
				all_refs[ref] = nil
				images = append(images, ref)
			}
		}
	}

	return images
}

func getImageMetadata(ref string) (numLayers int, totalSize int64, err error) {
	imgRef, err := name.ParseReference(ref)
	if err != nil {
		return 0, 0, err
	}

	img, err := remote.Image(imgRef)
	if err != nil {
		return 0, 0, err
	}

	layers, err := img.Layers()
	if err != nil {
		return 0, 0, err
	}

	var size int64
	for _, layer := range layers {
		sz, err := layer.Size()
		if err != nil {
			return 0, 0, err
		}
		size += sz
	}

	return len(layers), size, nil
}

func main() {
	f, err := os.Create("images.json")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = f.Close()
	}()

	images := fetchImages()
	fmt.Printf("Found %d images", len(images))
	// imgMeta := make([]ImageMetadata, 0, len(images))
	// for idx, ref := range images {
	// 	fmt.Printf("Getting metadata for %d: %s", idx, ref)
	// 	numLayers, totalSize, err := getImageMetadata(ref)
	// 	if err != nil {
	// 		// Log or skip
	// 		continue
	// 	}
	// 	imgMeta = append(imgMeta, ImageMetadata{
	// 		Name:      ref,
	// 		NumLayers: numLayers,
	// 		TotalSize: totalSize,
	// 	})
	// }


	contents, err := json.Marshal(images)
	if err != nil {
		panic(err)
	}

	_, err = f.Write(contents)
	if err != nil {
		panic(err)
	}
}
