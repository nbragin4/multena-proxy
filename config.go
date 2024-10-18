package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Log struct {
		Level     int  `mapstructure:"level"`
		LogTokens bool `mapstructure:"log_tokens"`
	} `mapstructure:"log"`

	Web struct {
		ProxyPort           int    `mapstructure:"proxy_port"`
		MetricsPort         int    `mapstructure:"metrics_port"`
		Host                string `mapstructure:"host"`
		TLSVerifySkip       bool   `mapstructure:"tls_verify_skip"`
		TrustedRootCaPath   string `mapstructure:"trusted_root_ca_path"`
		LabelStoreKind      string `mapstructure:"label_store_kind"`
		JwksCertURL         string `mapstructure:"jwks_cert_url"`
		OAuthGroupName      string `mapstructure:"oauth_group_name"`
		ServiceAccountToken string `mapstructure:"service_account_token"`
	} `mapstructure:"web"`

	Admin struct {
		Bypass bool   `mapstructure:"bypass"`
		Group  string `mapstructure:"group"`
	} `mapstructure:"admin"`

	Dev struct {
		Enabled  bool   `mapstructure:"enabled"`
		Username string `mapstructure:"username"`
	} `mapstructure:"dev"`

	Db struct {
		Enabled      bool   `mapstructure:"enabled"`
		User         string `mapstructure:"user"`
		PasswordPath string `mapstructure:"password_path"`
		Host         string `mapstructure:"host"`
		Port         int    `mapstructure:"port"`
		DbName       string `mapstructure:"dbName"`
		Query        string `mapstructure:"query"`
		TokenKey     string `mapstructure:"token_key"`
	} `mapstructure:"db"`

	Thanos struct {
		URL          string            `mapstructure:"url"`
		TenantLabel  string            `mapstructure:"tenant_label"`
		UseMutualTLS bool              `mapstructure:"use_mutual_tls"`
		Cert         string            `mapstructure:"cert"`
		Key          string            `mapstructure:"key"`
		Headers      map[string]string `mapstructure:"headers"`
	} `mapstructure:"thanos"`
	Loki struct {
		URL          string            `mapstructure:"url"`
		TenantLabel  string            `mapstructure:"tenant_label"`
		UseMutualTLS bool              `mapstructure:"use_mutual_tls"`
		Cert         string            `mapstructure:"cert"`
		Key          string            `mapstructure:"key"`
		Headers      map[string]string `mapstructure:"headers"`
	} `mapstructure:"loki"`
}

func (a *App) WithConfig() *App {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/config/config/")
	v.AddConfigPath("./configs")
	err := v.MergeInConfig()
	if err != nil {
		return nil
	}
	a.Cfg = &Config{}
	err = v.Unmarshal(a.Cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while unmarshalling config file")
	}
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().Str("file", e.Name).Msg("Config file changed")
		err := v.Unmarshal(a.Cfg)
		if err != nil {
			log.Error().Err(err).Msg("Error while unmarshalling config file")
			a.healthy = false
		}
		zerolog.SetGlobalLevel(zerolog.Level(a.Cfg.Log.Level))
	})
	v.WatchConfig()
	zerolog.SetGlobalLevel(zerolog.Level(a.Cfg.Log.Level))
	log.Debug().Any("config", a.Cfg).Msg("")
	return a
}

func (a *App) WithSAT() *App {
	if a.Cfg.Dev.Enabled {
		a.ServiceAccountToken = a.Cfg.Web.ServiceAccountToken
		return a
	}
	sa, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Fatal().Err(err).Msg("Error while reading service account token")
	}
	a.ServiceAccountToken = string(sa)
	return a
}

func (a *App) WithTLSConfig() *App {
	caCert, err := os.ReadFile("/etc/ssl/ca/ca-certificates.crt")
	if err != nil {
		log.Fatal().Err(err).Msg("Error while reading CA certificate")
	}
	log.Trace().Bytes("caCert", caCert).Msg("")

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
		log.Fatal().Msg("Failed to append CA certificate")
	}
	log.Debug().Any("rootCAs", rootCAs).Msg("")

	if a.Cfg.Web.TrustedRootCaPath != "" {
		err := filepath.Walk(a.Cfg.Web.TrustedRootCaPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || strings.Contains(info.Name(), "..") {
				return nil
			}

			certs, err := os.ReadFile(path)
			if err != nil {
				log.Error().Err(err).Msg("Error while reading trusted CA")
				return err
			}
			log.Debug().Str("path", path).Msg("Adding trusted CA")
			certs = append(certs, []byte("\n")...)
			rootCAs.AppendCertsFromPEM(certs)

			return nil
		})
		if err != nil {
			log.Error().Err(err).Msg("Error while traversing directory")
		}
	}

	var certificates []tls.Certificate

	lokiCert, err := tls.LoadX509KeyPair(a.Cfg.Loki.Cert, a.Cfg.Loki.Key)
	if err != nil {
		log.Error().Err(err).Msg("Error while loading loki certificate")
	} else {
		log.Debug().Str("path", a.Cfg.Loki.Cert).Msg("Adding Loki certificate")
		certificates = append(certificates, lokiCert)
	}

	thanosCert, err := tls.LoadX509KeyPair(a.Cfg.Thanos.Cert, a.Cfg.Thanos.Key)
	if err != nil {
		log.Error().Err(err).Msg("Error while loading thanos certificate")
	} else {
		log.Debug().Str("path", a.Cfg.Thanos.Cert).Msg("Adding Thanos certificate")
		certificates = append(certificates, thanosCert)
	}

	config := &tls.Config{
		InsecureSkipVerify: a.Cfg.Web.TLSVerifySkip,
		RootCAs:            rootCAs,
		Certificates:       certificates,
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = config
	return a
}

func (a *App) WithJWKS() *App {
	log.Info().Msg("Init JWKS config")

	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{a.Cfg.Web.JwksCertURL}) // Context is used to end the refresh goroutine.
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create a keyfunc from the server's URL")
	}
	log.Info().Str("url", a.Cfg.Web.JwksCertURL).Msg("JWKS URL")
	a.Jwks = jwks
	return a
}
