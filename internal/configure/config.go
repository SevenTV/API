package configure

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func checkErr(err error) {
	if err != nil {
		zap.S().Fatalw("config",
			"error", err,
		)
	}
}

func New() *Config {
	initLogging("info")

	config := viper.New()

	// Default config
	b, _ := json.Marshal(Config{
		ConfigFile: "config.yaml",
	})
	tmp := viper.New()
	defaultConfig := bytes.NewReader(b)
	tmp.SetConfigType("json")
	checkErr(tmp.ReadConfig(defaultConfig))
	checkErr(config.MergeConfigMap(viper.AllSettings()))

	pflag.String("config", "config.yaml", "Config file location")
	pflag.Bool("noheader", false, "Disable the startup header")

	pflag.Parse()
	checkErr(config.BindPFlags(pflag.CommandLine))

	// File
	config.SetConfigFile(config.GetString("config"))
	config.AddConfigPath(".")
	if err := config.ReadInConfig(); err == nil {
		checkErr(config.MergeInConfig())
	}

	bindEnvs(config, Config{})

	// Environment
	config.AutomaticEnv()
	config.SetEnvPrefix("API")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AllowEmptyEnv(true)

	// Print final config
	c := &Config{}
	checkErr(config.Unmarshal(&c))

	initLogging(c.Level)

	return c
}

func bindEnvs(config *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			bindEnvs(config, v.Interface(), append(parts, tv)...)
		default:
			_ = config.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

func BindEnvs(config *viper.Viper, iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			BindEnvs(config, v.Interface(), append(parts, tv)...)
		default:
			_ = config.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

type Config struct {
	Level         string `mapstructure:"level" json:"level"`
	ConfigFile    string `mapstructure:"config" json:"config"`
	NoHeader      bool   `mapstructure:"noheader" json:"noheader"`
	WebsiteURL    string `mapstructure:"website_url" json:"website_url"`
	OldWebsiteURL string `mapstructure:"website_old_url" json:"website_old_url"`
	NodeName      string `mapstructure:"node_name" json:"node_name"`
	TempFolder    string `mapstructure:"temp_folder" json:"temp_folder"`
	CdnURL        string `mapstructure:"cdn_url" json:"cdn_url"`

	Redis struct {
		Username   string   `mapstructure:"username" json:"username"`
		Password   string   `mapstructure:"password" json:"password"`
		Database   int      `mapstructure:"db" json:"db"`
		Sentinel   bool     `mapstructure:"sentinel" json:"sentinel"`
		Addresses  []string `mapstructure:"addresses" json:"addresses"`
		MasterName string   `mapstructure:"master_name" json:"master_name"`
	} `mapstructure:"redis" json:"redis"`

	Mongo struct {
		URI    string `mapstructure:"uri" json:"uri"`
		DB     string `mapstructure:"db" json:"db"`
		Direct bool   `mapstructure:"direct" json:"direct"`
	} `mapstructure:"mongo" json:"mongo"`

	Health struct {
		Enabled bool
		Bind    string
	}

	Monitoring struct {
		Enabled bool
		Bind    string
		Labels  Labels
	}

	Http struct {
		Addr          string `mapstructure:"uri" json:"uri"`
		VersionSuffix string `mapstructure:"version_suffix" json:"version_suffix"`
		Ports         struct {
			GQL  int `mapstructure:"gql" json:"gql"`
			REST int `mapstructure:"rest" json:"rest"`
		} `mapstructure:"ports" json:"ports"`

		Type             string `mapstructure:"type" json:"type"`
		OauthRedirectURI string `mapstructure:"oauth_redirect_uri" json:"oauth_redirect_uri"`
		Cookie           struct {
			Domain string `mapstructure:"cookie_domain" json:"cookie_domain"`
			Secure bool   `mapstructure:"cookie_secure" json:"cookie_secure"`
		}
		Quota struct {
			DefaultLimit  int32 `mapstructure:"quota_default_limit" json:"quota_default_limit"`
			MaxBadQueries int64 `mapstructure:"quota_max_bad_queries" json:"quota_max_bad_queries"`
		}
	} `mapstructure:"http" json:"http"`

	Platforms struct {
		Twitch struct {
			ClientID     string `mapstructure:"client_id" json:"client_id"`
			ClientSecret string `mapstructure:"client_secret" json:"client_secret"`
			RedirectURI  string `mapstructure:"redirect_uri" json:"redirect_uri"`
		} `mapstructure:"twitch" json:"twitch"`
	} `mapstructure:"platforms" json:"platforms"`

	RMQ struct {
		URI                            string `mapstructure:"uri" json:"uri"`
		ImageProcessorJobsQueueName    string `mapstructure:"image_processor_jobs_queue_name" json:"image_processor_jobs_queue_name"`
		ImageProcessorResultsQueueName string `mapstructure:"image_processor_results_queue_name" json:"image_processor_results_queue_name"`
	} `mapstructure:"rmq" json:"rmq"`

	S3 struct {
		AccessToken string `mapstructure:"access_token" json:"access_token"`
		SecretKey   string `mapstructure:"secret_key" json:"secret_key"`
		Region      string `mapstructure:"region" json:"region"`
		Bucket      string `mapstructure:"bucket" json:"bucket"`
		Endpoint    string `mapstructure:"endpoint" json:"endpoint"`
	} `mapstructure:"s3" json:"s3"`

	Auth struct {
		Platforms []struct {
			Name    string `mapstructure:"name" json:"name"`
			Enabled bool   `mapstructure:"enabled" json:"enabled"`
		} `mapstructure:"platforms" json:"platforms"`
	} `mapstructure:"auth" json:"auth"`

	Credentials struct {
		JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret"`
	} `mapstructure:"credentials" json:"credentials"`
}

type Labels []struct {
	Key   string `mapstructure:"key" json:"key"`
	Value string `mapstructure:"value" json:"value"`
}

func (l Labels) ToPrometheus() prometheus.Labels {
	mp := prometheus.Labels{}

	for _, v := range l {
		mp[v.Key] = v.Value
	}

	return mp
}
