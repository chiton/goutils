package aws

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/testcat"
)

type TestConfig struct {
	BrokerAddr     string `yaml:"broker-addr" env:"BROKER_ADDR" env-default:"amqp://localhost:5672"`
	BrokerUsername string `yaml:"broker-username" env:"BROKER_USERNAME" env-default:"admin"`
	BrokerPassword string `yaml:"broker-password" env:"BROKER_PASSWORD" env-default:""`
	MongoURL       string `yaml:"mongo-url" env:"MONGO_URL" env-default:"mongodb://localhost:27024"`
	MongoUser      string `yaml:"mongo-user" env:"MONGO_USER" env-default:"admin"`
	MongoPassword  string `yaml:"mongo-password" env:"MONGO_PASSWORD" env-default:""`
}

func (c *TestConfig) Validate() error {
	return nil
}

func (c *TestConfig) Strings() []string {
	var builder []string

	builder = append(builder,
		fmt.Sprintf("BrokerAddr: %s", c.BrokerAddr),
		fmt.Sprintf("BrokerUsername: %s", c.BrokerUsername),
		fmt.Sprintf("BrokerPassword: %s", "*****"), // intentionally scrubbed
		fmt.Sprintf("MongoURL: %s", c.MongoURL),
		fmt.Sprintf("MongoUser: %s", c.MongoUser),
		fmt.Sprintf("MongoPassword: %s", "*****"), // intentionally scrubbed
	)

	return builder
}

func TestAwsAppConfig(t *testing.T) {
	testcat.CheckTestCategory(t, testcat.Integration)

	// this is to make up for the fact that this test could be running on either the
	// prod or non-prod accounts in AWS. We're making the non-prod account the exception.
	// The env var for this test is the CICD file, integration-test.yaml
	if os.Getenv(Application) == "" {
		os.Setenv(Application, "dev-sas-apim-api")
	}

	os.Setenv(Region, "us-west-2")
	os.Setenv(Profile, "default")
	os.Setenv(Env, "default")

	awsCfg, err := NewProvider[*TestConfig](MustGetEnvs())

	require.NoError(t, err, "could not create Provider")

	cfg := &TestConfig{}
	err = awsCfg.GetConfig(context.Background(), cfg)
	require.NoError(t, err, "could not load config from AWS -- if you're running locally, make sure you are logged in to AWS CLI")

	require.NotEmpty(t, cfg.BrokerAddr)
	require.NotEmpty(t, cfg.MongoURL)
}
