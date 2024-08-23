package layer



import (

	"bytes"

	"encoding/json"

	"os"

)



type LayerMetadata struct {

	Size  int64  `json:"size"`

	Layer string `json:"layer"`

}



type ImageMetadata struct {

	Id              string          `json:"id"`

	Name            string          `json:"name"`

	NameWithoutRepo string          `json:"name_without_repo"`

	Tag             string          `json:"tag"`

	TotalSize       int64           `json:"total_size"`

	LayerMetadata   []LayerMetadata `json:"layer_metadata"`

}



type ImageMetadataLists struct {

	CatchFile string

	Lists     map[string]ImageMetadata

}



func NewImageMetadataListFromCache(filePath string) (*ImageMetadataLists, error) {

	jf, err := NewJsonFile(filePath)

	if err != nil {

		return &ImageMetadataLists{}, err

	}

	res := &ImageMetadataLists{}

	_, err = jf.Load(res)

	res.CatchFile = filePath

	return res, err

}



func (re *ImageMetadataLists) GetAllKnownLayers() []LayerMetadata {

	res := []LayerMetadata{}

	for _, mt := range re.Lists {

		for _, layerStr := range mt.LayerMetadata {

			res = append(res, layerStr)

		}

	}

	return res

}



func (re *ImageMetadataLists) Dump(filePath string) error {

	jf, err := NewJsonFile(filePath)

	if err != nil {

		return err

	}

	return jf.Dump(&re)

}



func (re *ImageMetadataLists) Fromat() (bytes.Buffer, error) {

	var str bytes.Buffer

	b, err := json.Marshal(re)



	if err != nil {

		return str, err

	}

	_ = json.Indent(&str, b, "", "     ")

	return str, nil

}



func (re *ImageMetadataLists) Search(image DockerImageName) (ImageMetadata, error) {

	res, ok := re.Lists[image.NameWithoutRepoAddr()]

	if ok {

		return res, nil

	}

	return res, os.ErrNotExist

}



func (re *ImageMetadataLists) SearchLayer(layer string) int64 {

	allLayer := re.GetAllKnownLayers()

	for _, l := range allLayer {

		if l.Layer == layer {

			return l.Size

		}

	}

	return 0

}



func ComputeLayerSize(metadata []LayerMetadata) int64 {

	res := int64(0)

	for _, data := range metadata {

		res += data.Size

	}

	return res

}


