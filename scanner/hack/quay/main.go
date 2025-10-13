package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

func fetchImages() []string {
	fetchRepos := func(page int) []*Repository {
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
	for i := 0; i < 25; i++ {
		repos := fetchRepos(i)
		for _, repo := range repos {
			tags := fetchTags(repo)
			for _, tag := range tags {
				images = append(images, fmt.Sprintf("quay.io/%s/%s:%s", repo.Namespace.Name, repo.Name, tag))
			}
		}
	}

	return images
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

	contents, err := json.Marshal(images)
	if err != nil {
		panic(err)
	}

	_, err = f.Write(contents)
	if err != nil {
		panic(err)
	}
}
