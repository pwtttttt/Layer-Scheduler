package layer

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"k8s.io/klog/v2"
)

type Registry struct {
	catchFile string
	Registry  *registry.Registry
}

func NewRegistry(url, username, password string) (*Registry, error) {
	reg, err := registry.New(url, username, password)
	if err != nil {
		return nil, err
	}
	return &Registry{
		Registry: reg,
	}, nil
}
func (r *Registry) ListRepositories() ([]string, error) {
	return r.Registry.Repositories()
}
func (r *Registry) ListImageTags(image string) ([]string, error) {
	return r.Registry.Tags(image)
}
func (r *Registry) GetLens() (int, int, error) {
	registry, err := r.ListRepositories()
	if err != nil {
		return 0, 0, err
	}
	tagLen := 0
	for _, reg := range registry {
		tags, err := r.ListImageTags(reg)
		if err != nil {
			return 0, 0, err
		}
		tagLen += len(tags)
	}
	return len(registry), tagLen, nil
}
func (r *Registry) GetImageLayers(image, tag string) (*schema2.DeserializedManifest, error) {
	manifest, err := r.Registry.ManifestV2(image, tag)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}
func (r *Registry) GetImageMetadata(img DockerImageName) (*ImageMetadataLists, error) {
	res := &ImageMetadataLists{}
	var imgList = map[string]ImageMetadata{}
	layer, err := r.GetImageLayers(img.NameWithoutRepoAddr(), img.Tag())
	if err != nil {
		return res, err
	}
	l := []LayerMetadata{}
	totalSize := int64(0)
	for _, lr := range layer.Manifest.Layers {
		lm := LayerMetadata{
			Size:  lr.Size,
			Layer: string(lr.Digest),
		}
		totalSize += lr.Size
		l = append(l, lm)
	}
	imgList[img.String()] = ImageMetadata{
		Name:            img.Name(),
		NameWithoutRepo: img.NameWithoutRepoAddr(),
		Tag:             img.Tag(),
		TotalSize:       totalSize,
		Id:              layer.Config.Digest.Encoded(),
		LayerMetadata:   l,
	}
	res.Lists = imgList
	return res, nil
}
func (r *Registry) GetLocalImageLayers(images []DockerImageName) (*ImageMetadataLists, error) {
	res := &ImageMetadataLists{}
	var imgList = map[string]ImageMetadata{}
	for _, img := range images {
		layer, err := r.GetImageLayers(img.NameWithoutRepoAddr(), img.Tag())
		if err != nil {
			return res, err
		}
		l := []LayerMetadata{}
		totalSize := int64(0)
		for _, lr := range layer.Manifest.Layers {
			lm := LayerMetadata{
				Size:  lr.Size,
				Layer: string(lr.Digest),
			}
			totalSize += lr.Size
			l = append(l, lm)
		}
		imgList[img.String()] = ImageMetadata{
			Name:            img.Name(),
			NameWithoutRepo: img.NameWithoutRepoAddr(),
			Tag:             img.Tag(),
			TotalSize:       totalSize,
			Id:              layer.Config.Digest.Encoded(),
			LayerMetadata:   l,
		}
	}
	res.Lists = imgList
	return res, nil
}
func (r *Registry) GetRemoteImageLayers() (*ImageMetadataLists, error) {
	var res = &ImageMetadataLists{}
	var imgList = map[string]ImageMetadata{}
	repos, err := r.ListRepositories()
	if err != nil {
		return res, err
	}
	for _, repo := range repos {
		tags, err := r.ListImageTags(repo)
		if err != nil {
			return res, err
		}
		for _, tag := range tags {
			layer, err := r.GetImageLayers(repo, tag)
			if err != nil {
				return res, err
			}
			l := []LayerMetadata{}
			for _, lr := range layer.Manifest.Layers {
				lm := LayerMetadata{
					Size:  lr.Size,
					Layer: string(lr.Digest),
				}
				l = append(l, lm)
			}
			imgList[repo+":"+tag] = ImageMetadata{
				Name:            repo,
				NameWithoutRepo: repo,
				Tag:             tag,
				TotalSize:       layer.Config.Size,
				Id:              layer.Config.Digest.Encoded(),
				LayerMetadata:   l,
			}
		}
	}
	res.Lists = imgList
	return res, nil
}
func (r *Registry) CreateCatch(filePath string) error {
	imginfo, err := r.GetRemoteImageLayers()
	if err != nil {
		klog.Errorf("读取本地镜像层信息失败, err: %s", err)
		return err
	}
	return imginfo.Dump(filePath)
}
func (r *Registry) Watcher(wait time.Duration, filePath string, ctx context.Context) {
	oldRepoLen, oldTagLen, err := r.GetLens()
	if err != nil {
		klog.Errorf("未获取到镜像, err: %v", err)
	}
	defer func() {
		fmt.Println("监听器退出")
	}()
	if !Exists(filePath) {
		klog.Infof("未找到本地缓存文件%s, 生成缓存", filePath)
	}
	imginfo, err := r.GetRemoteImageLayers()
	if err != nil {
		klog.Errorf("读取本地镜像层信息失败, err: %s", err)
	}
	err = imginfo.Dump(filePath)
	if err != nil {
		klog.Errorf("更新缓存失败, err: %s", err)
	}
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		default:
		}
		newRepoLen, newTagLen, err := r.GetLens()
		if err != nil {
			klog.Errorf("获取远程仓库数据失败, err: %s", err)
		}
		if newRepoLen > oldRepoLen || newTagLen > oldTagLen {
			klog.Infof("检测到镜像仓库变化，刷新本地缓存")
			imginfo, err := r.GetRemoteImageLayers()
			if err != nil {
				klog.Errorf("读取本地镜像层信息失败, err: %s", err)
			}
			err = imginfo.Dump(filePath)
			if err != nil {
				klog.Errorf("更新缓存失败, err: %s", err)
			}
		}
		time.Sleep(wait)
	}
}
