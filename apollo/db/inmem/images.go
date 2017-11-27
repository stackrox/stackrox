package inmem

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

func (i *InMemoryStore) loadImages() error {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	images, err := i.persistent.GetImages(&v1.GetImagesRequest{})
	if err != nil {
		return err
	}
	for _, image := range images {
		i.images[image.Sha] = image
	}
	return nil
}

// GetImages returns all images
func (i *InMemoryStore) GetImages(request *v1.GetImagesRequest) ([]*v1.Image, error) {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	images := make([]*v1.Image, 0, len(i.images))
	for _, image := range i.images {
		images = append(images, image)
	}
	sort.SliceStable(images, func(i, j int) bool { return images[i].Sha < images[j].Sha })
	return images, nil
}

func (i *InMemoryStore) insertImage(image *v1.Image) {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	i.images[image.Sha] = image
}

// AddImage adds an image to the database
func (i *InMemoryStore) AddImage(image *v1.Image) error {
	if err := i.persistent.AddImage(image); err != nil {
		return err
	}
	i.insertImage(image)
	return nil
}

// UpdateImage updates an image
func (i *InMemoryStore) UpdateImage(image *v1.Image) error {
	if err := i.persistent.UpdateImage(image); err != nil {
		return err
	}
	i.insertImage(image)
	return nil
}

// RemoveImage removes a specific image specified by it's SHA
func (i *InMemoryStore) RemoveImage(sha string) error {
	i.imageMutex.Lock()
	defer i.imageMutex.Unlock()
	if err := i.persistent.RemoveImage(sha); err != nil {
		return err
	}
	delete(i.images, sha)
	return nil
}
