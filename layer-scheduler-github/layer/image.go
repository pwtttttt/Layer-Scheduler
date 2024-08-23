package layer



import (

	"context"

	"strings"



	"github.com/docker/docker/api/types"

	"github.com/docker/docker/api/types/filters"

	"github.com/docker/docker/client"

)



type DockerImageName string



func (di DockerImageName) String() string {

	return string(di)

}



func (di DockerImageName) Name() string {

	return strings.Join(strings.Split(di.String(), ":")[:2], ":")

}



func (di DockerImageName) NameWithoutRepoAddr() string {

	return strings.Join(strings.Split(di.Name(), "/")[1:], "/")

}



func (di DockerImageName) Tag() string {

	return strings.Split(di.String(), ":")[2]

}



type DockerImages struct {

	Cli       *client.Client

	CatchFile string

}



func NewDockerImageLocal() (*DockerImages, error) {

	client, err := client.NewClientWithOpts(client.FromEnv)

	if err != nil {

		return nil, err

	}

	return &DockerImages{

		Cli: client,

	}, nil

}



func NewDockerImage(address string, catchFile string) (*DockerImages, error) {

	client, err := client.NewClientWithOpts(client.WithHost("tcp://"+address+":2375"), client.WithAPIVersionNegotiation())

	if err != nil {

		return nil, err

	}

	return &DockerImages{

		Cli:       client,

		CatchFile: catchFile,

	}, nil

}



func (d *DockerImages) ListAllLocalImagesInRepo(repo string) []DockerImageName {

	res := []DockerImageName{}

	r, _ := d.Cli.ImageList(context.TODO(), types.ImageListOptions{})

	for _, v := range r {

		for _, tag := range v.RepoTags {

			if strings.HasPrefix(tag, repo) {

				res = append(res, DockerImageName(tag))

				break

			}

		}

	}

	return res

}



func (d *DockerImages) CheckImageExistOnLocal(imageName string) (bool, error) {

	arg := filters.NewArgs(filters.KeyValuePair{

		Key:   "reference",

		Value: imageName,

	})

	images, err := d.Cli.ImageList(context.TODO(), types.ImageListOptions{

		All:     true,

		Filters: arg,

	})

	if err != nil || len(images) == 0 {

		return false, err

	}

	return true, nil

}



func (d *DockerImages) GetImageLayer(imageName string, handler *ImageMetadataLists) (ImageMetadata, error) {

	return handler.Search(DockerImageName(imageName))

}


