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

type MessageQueueMode string

const (
	MessageQueueModeRMQ = "RMQ"
	MessageQueueModeSQS = "SQS"
)

type Config struct {
	Level         string `mapstructure:"level" json:"level"`
	ConfigFile    string `mapstructure:"config" json:"config"`
	NoHeader      bool   `mapstructure:"noheader" json:"noheader"`
	WebsiteURL    string `mapstructure:"website_url" json:"website_url"`
	OldWebsiteURL string `mapstructure:"website_old_url" json:"website_old_url"`
	CdnURL        string `mapstructure:"cdn_url" json:"cdn_url"`

	K8S struct {
		NodeName string `mapstructure:"node_name" json:"node_name"`
		PodName  string `mapstructure:"pod_name" json:"pod_name"`
	} `mapstructure:"k8s" json:"k8s"`

	Redis struct {
		Username   string   `mapstructure:"username" json:"username"`
		Password   string   `mapstructure:"password" json:"password"`
		Database   int      `mapstructure:"db" json:"db"`
		Sentinel   bool     `mapstructure:"sentinel" json:"sentinel"`
		Addresses  []string `mapstructure:"addresses" json:"addresses"`
		MasterName string   `mapstructure:"master_name" json:"master_name"`
	} `mapstructure:"redis" json:"redis"`

	Mongo struct {
		URI      string `mapstructure:"uri" json:"uri"`
		Username string `mapstructure:"username" json:"username"`
		Password string `mapstructure:"password" json:"password"`
		DB       string `mapstructure:"db" json:"db"`
		Direct   bool   `mapstructure:"direct" json:"direct"`
	} `mapstructure:"mongo" json:"mongo"`

	Health struct {
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
		Bind    string `mapstructure:"bind" json:"bind"`
	} `mapstructure:"health" json:"health"`

	PProf struct {
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
		Bind    string `mapstructure:"bind" json:"bind"`
	} `mapstructure:"pprof" json:"pprof"`

	Monitoring struct {
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
		Bind    string `mapstructure:"bind" json:"bind"`
		Labels  Labels `mapstructure:"labels" json:"labels"`
	} `mapstructure:"monitoring" json:"monitoring"`

	EventBridge struct {
		Enabled bool   `mapstructure:"enabled" json:"enabled"`
		Bind    string `mapstructure:"bind" json:"bind"`
	} `mapstructure:"event_bridge" json:"event_bridge"`

	Chatterino struct {
		Version string `mapstructure:"version" json:"version"`
		Stable  struct {
			Win struct {
				Download         string `mapstructure:"download" json:"download"`
				PortableDownload string `mapstructure:"portable_download" json:"portable_download"`
				UpdateExe        string `mapstructure:"update_exe" json:"update_exe"`
			} `mapstructure:"win" json:"win"`
			Linux struct {
				Download         string `mapstructure:"download" json:"download"`
				PortableDownload string `mapstructure:"portable_download" json:"portable_download"`
				UpdateExe        string `mapstructure:"update_exe" json:"update_exe"`
			} `mapstructure:"linux" json:"linux"`
			Macos struct {
				Download         string `mapstructure:"download" json:"download"`
				PortableDownload string `mapstructure:"portable_download" json:"portable_download"`
				UpdateExe        string `mapstructure:"update_exe" json:"update_exe"`
			} `mapstructure:"macos" json:"macos"`
		} `mapstructure:"stable" json:"stable"`
		Beta struct {
			Win struct {
				Download         string `mapstructure:"download" json:"download"`
				PortableDownload string `mapstructure:"portable_download" json:"portable_download"`
				UpdateExe        string `mapstructure:"update_exe" json:"update_exe"`
			} `mapstructure:"win" json:"win"`
			Linux struct {
				Download         string `mapstructure:"download" json:"download"`
				PortableDownload string `mapstructure:"portable_download" json:"portable_download"`
				UpdateExe        string `mapstructure:"update_exe" json:"update_exe"`
			} `mapstructure:"linux" json:"linux"`
			Macos struct {
				Download         string `mapstructure:"download" json:"download"`
				PortableDownload string `mapstructure:"portable_download" json:"portable_download"`
				UpdateExe        string `mapstructure:"update_exe" json:"update_exe"`
			} `mapstructure:"macos" json:"macos"`
		} `mapstructure:"beta" json:"beta"`
	} `mapstructure:"chatterino" json:"chatterino"`

	Http struct {
		Addr          string `mapstructure:"addr" json:"addr"`
		VersionSuffix string `mapstructure:"version_suffix" json:"version_suffix"`
		Ports         struct {
			GQL    int `mapstructure:"gql" json:"gql"`
			REST   int `mapstructure:"rest" json:"rest"`
			Bridge int `mapstructure:"bridge" json:"bridge"`
		} `mapstructure:"ports" json:"ports"`

		DisablePresences   bool `mapstructure:"disable_presences" json:"disable_presences"`
		DisableEventBridge bool `mapstructure:"disable_event_bridge" json:"disable_event_bridge"`

		ProxiedEndpoint struct {
			URL         string `mapstructure:"url" json:"url"`
			BypassToken string `mapstructure:"bypass_token" json:"bypass_token"`
		} `mapstructure:"proxied_endpoint" json:"proxied_endpoint"`

		Cookie struct {
			Domain    string   `mapstructure:"domain" json:"domain"`
			Secure    bool     `mapstructure:"secure" json:"secure"`
			Whitelist []string `mapstructure:"whitelist" json:"whitelist"`
		} `mapstructure:"cookie" json:"cookie"`
	} `mapstructure:"http" json:"http"`

	Platforms PlatformConfig `mapstructure:"platforms" json:"platforms"`

	Limits struct {
		MaxPage int `mapstructure:"max_page" json:"max_page"`

		Buckets struct {
			GQL2            [2]int64 `mapstructure:"gql_v2" json:"gql_v2"`
			GQL3            [2]int64 `mapstructure:"gql_v3" json:"gql_v3"`
			ImageProcessing [2]int64 `mapstructure:"image_processing" json:"image_processing"`
		} `mapstructure:"buckets" json:"buckets"`

		Quota struct {
			DefaultLimit         int32 `mapstructure:"default_limit" json:"default_limit"`
			MaxBadQueries        int64 `mapstructure:"max_bad_queries" json:"max_bad_queries"`
			MaxActiveModRequests int64 `mapstructure:"max_active_mod_requests" json:"max_active_mod_requests"`
		} `mapstructure:"quota" json:"quota"`

		Emotes struct {
			MaxProcessingTimeSeconds int      `mapstructure:"max_processing_time_seconds" json:"max_processing_time_seconds"`
			MaxWidth                 int      `mapstructure:"max_width" json:"max_width"`
			MaxHeight                int      `mapstructure:"max_height" json:"max_height"`
			MaxFrameCount            int      `mapstructure:"max_frame_count" json:"max_frame_count"`
			MaxTags                  int      `mapstructure:"max_tags" json:"max_tags"`
			ReservedTags             []string `mapstructure:"reserved_tags" json:"reserved_tags"`
		} `mapstructure:"emotes" json:"emotes"`
	} `mapstructure:"limits" json:"limits"`

	MessageQueue struct {
		Mode MessageQueueMode `mapstructure:"mode" json:"mode"`

		ImageProcessorJobsQueueName                string `mapstructure:"image_processor_jobs_queue_name" json:"image_processor_jobs_queue_name"`
		ImageProcessorResultsQueueName             string `mapstructure:"image_processor_results_queue_name" json:"image_processor_results_queue_name"`
		ImageProcessorUserPicturesResultsQueueName string `mapstructure:"image_processor_user_pictures_results_queue_name" json:"image_processor_user_pictures_results_queue_name"`

		RMQ struct {
			URI                  string `mapstructure:"uri" json:"uri"`
			MaxReconnectAttempts int    `mapstructure:"max_reconnect_attempts" json:"max_reconnect_attempts"`
		} `mapstructure:"rmq" json:"rmq"`

		SQS struct {
			Region           string `mapstructure:"region" json:"region"`
			AccessToken      string `mapstructure:"access_token" json:"access_token"`
			SecretKey        string `mapstructure:"secret_key" json:"secret_key"`
			MaxRetryAttempts int    `mapstructure:"max_retry_attempts" json:"max_retry_attempts"`
		} `mapstructure:"sqs" json:"sqs"`
	} `mapstructure:"message_queue" json:"message_queue"`

	S3 struct {
		Enabled        bool   `mapstructure:"enabled" json:"enabled"`
		AccessToken    string `mapstructure:"access_token" json:"access_token"`
		SecretKey      string `mapstructure:"secret_key" json:"secret_key"`
		Region         string `mapstructure:"region" json:"region"`
		InternalBucket string `mapstructure:"internal_bucket" json:"internal_bucket"`
		PublicBucket   string `mapstructure:"public_bucket" json:"public_bucket"`
		Endpoint       string `mapstructure:"endpoint" json:"endpoint"`
		Namespace      string `mapstructure:"namespace" json:"namespace"`
	} `mapstructure:"s3" json:"s3"`

	Credentials struct {
		JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret"`
	} `mapstructure:"credentials" json:"credentials"`
}

type PlatformConfig struct {
	Twitch struct {
		ClientID     string `mapstructure:"client_id" json:"client_id"`
		ClientSecret string `mapstructure:"client_secret" json:"client_secret"`
		RedirectURI  string `mapstructure:"redirect_uri" json:"redirect_uri"`
	} `mapstructure:"twitch" json:"twitch"`
	YouTube struct {
		APIKey       string `mapstructure:"api_key" json:"api_key"`
		ClientID     string `mapstructure:"client_id" json:"client_id"`
		ClientSecret string `mapstructure:"client_secret" json:"client_secret"`
		RedirectURI  string `mapstructure:"redirect_uri" json:"redirect_uri"`
	} `mapstructure:"youtube" json:"youtube"`
	Discord struct {
		ClientID     string `mapstructure:"client_id" json:"client_id"`
		ClientSecret string `mapstructure:"client_secret" json:"client_secret"`
		RedirectURI  string `mapstructure:"redirect_uri" json:"redirect_uri"`

		API      string            `mapstructure:"api" json:"api"`
		Channels map[string]string `mapstructure:"channels" json:"channels"`
	} `mapstructure:"discord" json:"discord"`
	Kick struct {
		ChallengeToken string `mapstructure:"challenge_token" json:"challenge_token"`
	} `mapstructure:"kick" json:"kick"`
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
