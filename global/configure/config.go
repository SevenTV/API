package configure

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func checkErr(err error) {
	if err != nil {
		logrus.WithError(err).Fatal("config")
	}
}

func New() *Config {
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
	err := config.ReadInConfig()
	if err != nil {
		logrus.Warning(err)
		logrus.Info("Using default config")
	} else {
		checkErr(config.MergeInConfig())
	}

	BindEnvs(config, Config{})

	// Environment
	config.AutomaticEnv()
	config.SetEnvPrefix("GQL")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AllowEmptyEnv(true)

	// Print final config
	c := &Config{}
	checkErr(config.Unmarshal(&c))

	initLogging(c.Level)

	return c
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
	Level      string `mapstructure:"level" json:"level"`
	ConfigFile string `mapstructure:"config" json:"config"`
	NoHeader   bool   `mapstructure:"noheader" json:"noheader"`
	WebsiteURL string `mapstructure:"website_url" json:"website_url"`
	NodeName   string `mapstructure:"node_name" json:"node_name"`
	CdnURL     string `mapstructure:"cdn_url" json:"cdn_url"`

	Redis struct {
		URI      string `mapstructure:"uri" json:"uri"`
		Username string `mapstructure:"username" json:"username"`
		Password string `mapstructure:"password" json:"password"`
		Database int    `mapstructure:"db" json:"db"`
	} `mapstructure:"redis" json:"redis"`

	Mongo struct {
		URI string `mapstructure:"uri" json:"uri"`
		DB  string `mapstructure:"db" json:"db"`
	} `mapstructure:"mongo" json:"mongo"`

	Http struct {
		Addr  string `mapstructure:"uri" json:"uri"`
		Ports struct {
			GQL  int `mapstructure:"gql" json:"gql"`
			REST int `mapstructure:"rest" json:"rest"`
		} `mapstructure:"ports" json:"ports"`
		Type               string `mapstructure:"type" json:"type"`
		OauthRedirectURI   string `mapstructure:"oauth_redirect_uri" json:"oauth_redirect_uri"`
		QuotaDefaultLimit  int32  `mapstructure:"quota_default_limit" json:"quota_default_limit"`
		QuotaMaxBadQueries int64  `mapstructure:"quota_max_bad_queries" json:"quota_max_bad_queries"`
	} `mapstructure:"http" json:"http"`

	Auth struct {
		Secret string `mapstructure:"secret" json:"secret"`

		Platforms []struct {
			Name    string `mapstructure:"name" json:"name"`
			Enabled bool   `mapstructure:"enabled" json:"enabled"`
		} `mapstructure:"platforms" json:"platforms"`
	} `mapstructure:"auth" json:"auth"`

	Credentials struct {
		JWTSecret string `mapstructure:"jwt_secret" json:"jwt_secret"`
	} `mapstructure:"credentials" json:"credentials"`
}
