# goutils

Misc utilities that are too small to put in their own individual repos.

To install:

```go
go get gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils
```

## Packages

The following are the packages included in this module along with a brief
description of their usage.

### Package `commander`

This package is used to create a more dynamic `switch`/`case`. Handlers are registered
with the commander and then executed by sending commands. This is most useful when
implementing a Rest API against a DDD-oriented application where commands are
a key aspect of its architecture. See [api-management](https://gitlab.edgecastcdn.net/edgecast/web-platform/identity/api-management)
repo for an example usage.

### Package `config`

This package is used to help load configuration files from either a .yml file or from
AWS App Config.

```go
import gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/config
```

If loading from App Config, there is also the option of loading secrets
from AWS Secrets Manager pulled through Parameter Store by using the `WithParamStoreTransform()`
func.


Here's an example of loading secrets with a transform.

```go
func LoadAwsConfigProvider[T config.Config]() (*awsConfig.Provider[T], error) {
	awsProvider, err := awsConfig.NewProvider[T](awsConfig.MustGetEnvs())
	if err != nil {
		return nil, err
	}

	kafkaKey := os.Getenv(ParamStoreSecretKafka)

	awsProvider.WithParamStoreTransform(kafkaKey, func(from string) (string, error) {
		fromMap := map[string]any{}
		toMap := map[string]any{}

		err := json.Unmarshal([]byte(from), &fromMap)
		if err != nil {
			return "", err
		}

		toMap["broker-username"] = fromMap["username"]
		toMap["broker-password"] = fromMap["password"]

		to, err := yaml.Marshal(toMap)
		if err != nil {
			return "", err
		}

		return string(to), nil
	})

	return awsProvider, nil
}
```

### Package `correlation`

This package is used to help with getting and setting correlation ids in the `context`.

Import using this:

```go
import gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/correlation
```

### Package `environ`

This package is used to support a container running in ECS.

Import using this:

```go
import gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/environ
```

This package is dependent on the environment variable `ECS_CONTAINER_METADATA_URI_V4`
being present. This is only available when the container is running on ECS, although
you can spoof this locally as well. See the [AWS ECS documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html)
on this subject for more information.

Two main functions here:
* `IsECS()` - returns true if the running environment is in ECS
* `GetECSMetadataURI()` - gets the URI from the `ECS_CONTAINER_METADATA_URI_V4` env var

### Package `log`

This package is used for creating new sugared loggers based `zap`. There are also
convenience functions that help with getting and setting the logger from a `context`.

Import using this: 

```go
import gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log
```

### Package `testcat`

This package helps with detecting which test category is set when running `go test`.
The category is set using the `CATEGORY` environment variable. See the util.go file
for more info.

Import using this:

```go
import gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/testcat
```

* `CheckTestCategory()` - can be used to either run or skip the current test based on the current category