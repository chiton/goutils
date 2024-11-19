package aws

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/smithy-go/ptr"
	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"

	goutilsconfig "gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/config"
	goutilslog "gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
)

const (
	Region      = "APPCONFIG_REGION"
	Application = "APPCONFIG_APPLICATION"
	Profile     = "APPCONFIG_PROFILE"
	Env         = "APPCONFIG_ENV"
)

type Provider[T goutilsconfig.Config] struct {
	application          string
	configProfile        string
	env                  string
	appConfigDataClient  AppConfigDataClient
	ssmClient            SsmClient
	paramStoreTransforms map[string]func(from string) (string, error)
}

type AppConfigDataClient interface {
	StartConfigurationSession(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error)
	GetLatestConfiguration(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error)
}

type SsmClient interface {
	GetParameters(ctx context.Context, params *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error)
}

// NewProvider create a config provider to read configuration from AWS AppConfig.
// Params:
//   - region - the region to pull the AppConfig and Parameters from. e.g. us-west-2
//   - application - the name of the AppConfig application
//   - profile - the name of the profile in the AppConfig application
//   - env - the name of the environment in the AppConfig application
//   - secretNames - the names of the secrets from Parameter Store to load and merge with the config in AppConfig
func NewProvider[T goutilsconfig.Config](
	region,
	application,
	profile,
	env string) (*Provider[T], error) {

	var zero Provider[T]

	if region == "" || application == "" || profile == "" || env == "" {
		return &zero, errors.New("region, application, profile, clientId and/or env was not provided -- check your environment variables")
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)

	if err != nil {
		return nil, err
	}

	appConfigDataClient := appconfigdata.NewFromConfig(awsConfig)
	ssmClient := ssm.NewFromConfig(awsConfig)

	return &Provider[T]{
		application:          application,
		configProfile:        profile,
		env:                  env,
		appConfigDataClient:  appConfigDataClient,
		ssmClient:            ssmClient,
		paramStoreTransforms: map[string]func(from string) (string, error){},
	}, nil
}

// MustGetEnvs grabs all the needed vars from the running environment to be able to create a new Provider with NewProvider().
// If any error occurs, it will fail with a panic.
// This makes it a very straightforward call:
//
// Example:
//
//	provider, err = awsConfig.NewProvider[apiserver.Config](awsConfig.MustGetEnvs())
func MustGetEnvs() (
	region string,
	application string,
	profile string,
	env string,
) {

	region = os.Getenv(Region)
	application = os.Getenv(Application)
	profile = os.Getenv(Profile)
	env = os.Getenv(Env)

	return
}

// WithParamStoreTransform helps to transform content that is in an AWS ParameterStore key into a target format
// or schema. This is needed because the content in the parameter store is not always in the format
// that you need it to be, so we have to transform it into a schema that is usable by the config provider.
// The given transform is invoked after a value at the Parameter Store Key has been retrieved and before
// the configs are overridden with the target value.
func (provider *Provider[T]) WithParamStoreTransform(key string, transform func(from string) (string, error)) {
	provider.paramStoreTransforms[key] = transform
}

func (provider *Provider[T]) GetConfig(ctx context.Context, cfg T) error {
	log := goutilslog.FromContext(ctx)

	var pollInterval int32 = 60

	startInput := &appconfigdata.StartConfigurationSessionInput{
		ApplicationIdentifier:                &provider.application,
		ConfigurationProfileIdentifier:       &provider.configProfile,
		EnvironmentIdentifier:                &provider.env,
		RequiredMinimumPollIntervalInSeconds: &pollInterval,
	}

	log.Infow("Loading config from AWS", "awsConfigInput", startInput)

	startOutput, err := provider.appConfigDataClient.StartConfigurationSession(ctx, startInput)

	if err != nil {
		return err
	}

	getLatestInput := &appconfigdata.GetLatestConfigurationInput{
		ConfigurationToken: startOutput.InitialConfigurationToken,
	}

	getLatestOutput, err := provider.appConfigDataClient.GetLatestConfiguration(ctx, getLatestInput)

	if err != nil {
		return err
	}

	log.Infof("Config loaded from AppConfig")

	rawConfig, err := provider.includeTransforms(ctx, getLatestOutput.Configuration)
	if err != nil {
		return err
	}

	// Load environment variables
	cleanenv.ReadEnv(cfg)

	err = yaml.Unmarshal(rawConfig, &cfg)
	if err != nil {
		panic("toConfig: error unmarshalling config from bytes")
	}

	return nil
}

func (provider *Provider[T]) includeTransforms(ctx context.Context, baseConfig []byte) ([]byte, error) {
	log := goutilslog.FromContext(ctx)

	if len(provider.paramStoreTransforms) == 0 {
		log.Info("No secrets were loaded because no Parameter Store transforms were given")
		return baseConfig, nil
	}

	configMap := map[string]any{}

	err := yaml.Unmarshal(baseConfig, configMap)
	if err != nil {
		log.Errorw("Failed to unmarshal base config", "error", err)
		return nil, err
	}

	var secretNames []string
	for key := range provider.paramStoreTransforms {
		secretNames = append(secretNames, key)
	}

	getInput := &ssm.GetParametersInput{
		Names:          secretNames,
		WithDecryption: ptr.Bool(true),
	}

	getOutput, err := provider.ssmClient.GetParameters(ctx, getInput)

	if err != nil {
		return nil, err
	}

	for _, param := range getOutput.Parameters {
		if param.Value != nil {
			transformed, err := provider.paramStoreTransforms[*param.Name](*param.Value)
			if err != nil {
				log.Errorw(fmt.Sprintf("Failed to transform secret %s", *param.Name), "error", err)
				continue
			}

			secretsMap := map[string]any{}

			err = yaml.Unmarshal([]byte(transformed), secretsMap)
			if err != nil {
				log.Errorw("Failed to unmarshal secret from Parameter Store: %s", *param.Name)
				return nil, err
			}

			mergeMaps(configMap, secretsMap)

			log.Infof("Merged secret from Parameter Store: %s", *param.Name)
		}
	}

	for _, param := range getOutput.InvalidParameters {
		log.Warnf("Invalid secret parameter could not be loaded: %s", param)
	}

	mergedResult, err := yaml.Marshal(configMap)
	if err != nil {
		log.Error("Failed to unmarshal merged config")
		return nil, err
	}

	return mergedResult, nil
}

// mergeMaps performs a top-level merge replacing or adding any fields from source and applying it to target.
// Only the target is modified.
func mergeMaps(target map[string]any, source map[string]any) {
	for sourceKey, sourceValue := range source {
		target[sourceKey] = sourceValue
	}
}
