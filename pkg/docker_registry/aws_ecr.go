package docker_registry

import (
	"fmt"
	"regexp"

	"github.com/google/go-containerregistry/pkg/name"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/flant/werf/pkg/image"
)

const AwsEcrImplementationName = "ecr"

var (
	awsEcrPatternRegexp = regexp.MustCompile(`^(\d{12})\.dkr\.ecr(-fips)?\.([a-zA-Z0-9][a-zA-Z0-9-_]*)\.amazonaws\.com(\.cn)?$`)
	awsEcrPatterns      = []string{awsEcrPatternRegexp.String()}
)

type awsEcr struct {
	*defaultImplementation
}

type awsEcrOptions struct {
	defaultImplementationOptions
}

func newAwsEcr(options awsEcrOptions) (*awsEcr, error) {
	d, err := newDefaultImplementation(options.defaultImplementationOptions)
	if err != nil {
		return nil, err
	}

	awsEcr := &awsEcr{defaultImplementation: d}

	return awsEcr, nil
}

func (r *awsEcr) DeleteRepoImage(repoImageList ...*image.Info) error {
	repositoriesByRegion := map[string]map[string][]*ecr.ImageIdentifier{}

	for _, repoImage := range repoImageList {
		_, region, repository, err := r.parseReference(repoImage.Repository)
		if err != nil {
			return err
		}

		imageIdentifiersByRepository, ok := repositoriesByRegion[region]
		if !ok {
			repositoriesByRegion[region] = map[string][]*ecr.ImageIdentifier{}
			imageIdentifiersByRepository = repositoriesByRegion[region]
		}

		imageIdentifiers, ok := imageIdentifiersByRepository[repository]
		if !ok {
			imageIdentifiers = []*ecr.ImageIdentifier{}
		}

		imageIdentifiers = append(imageIdentifiers, &ecr.ImageIdentifier{
			ImageDigest: &repoImage.RepoDigest,
		})

		repositoriesByRegion[region][repository] = imageIdentifiers
	}

	for region, imageIdentifiersByRepository := range repositoriesByRegion {
		mySession := session.Must(session.NewSession())
		service := ecr.New(mySession, aws.NewConfig().WithRegion(region))

		for repository, imageIdentifiers := range imageIdentifiersByRepository {
			_, err := service.BatchDeleteImage(&ecr.BatchDeleteImageInput{
				ImageIds:       imageIdentifiers,
				RepositoryName: &repository,
			})

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *awsEcr) CreateRepo(reference string) error {
	_, region, repository, err := r.parseReference(reference)
	if err != nil {
		return err
	}

	mySession := session.Must(session.NewSession())
	service := ecr.New(mySession, aws.NewConfig().WithRegion(region))

	if _, err := service.CreateRepository(&ecr.CreateRepositoryInput{
		ImageScanningConfiguration: nil,
		ImageTagMutability:         nil,
		RepositoryName:             &repository,
		Tags:                       nil,
	}); err != nil {
		return err
	}

	return nil
}

func (r *awsEcr) DeleteRepo(reference string) error {
	_, region, repository, err := r.parseReference(reference)
	if err != nil {
		return err
	}

	force := true

	mySession := session.Must(session.NewSession())
	service := ecr.New(mySession, aws.NewConfig().WithRegion(region))

	if _, err := service.DeleteRepository(&ecr.DeleteRepositoryInput{
		Force:          &force,
		RegistryId:     nil,
		RepositoryName: &repository,
	}); err != nil {
		return err
	}

	return nil
}

func (r *awsEcr) String() string {
	return AwsEcrImplementationName
}

func (r *awsEcr) parseReference(reference string) (string, string, string, error) {
	var registryId, region, repository string

	parsedReference, err := name.NewRepository(reference)
	if err != nil {
		return "", "", "", err
	}

	registryId, region, err = r.parseHostname(parsedReference.RegistryStr())
	if err != nil {
		return "", "", "", err
	}

	repository = parsedReference.RepositoryStr()

	return registryId, region, repository, nil
}

func (r *awsEcr) parseHostname(hostname string) (string, string, error) {
	var registryId, region string

	splitURL := awsEcrPatternRegexp.FindStringSubmatch(hostname)
	if len(splitURL) == 0 {
		return "", "", fmt.Errorf("%s is not a valid ECR repository URL", hostname)
	}

	registryId = splitURL[1]
	region = splitURL[3]

	return registryId, region, nil
}
