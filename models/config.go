package models

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/andygrunwald/go-jira"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Listen         string                      `yaml:"listen,omitempty"`
	TemplateDir    string                      `yaml:"templates_dir"`
	EnableSSL      bool                        `yaml:"enable_ssl,omitempty"`
	SSLCertificate tls.Certificate             `yaml:"-"`
	TLSPEM         TLSPem                      `yaml:"tls_pem,omitempty"`
	Log            Log                         `yaml:"log"`
	Jira           Jira                        `yaml:"jira"`
	DevMode        bool                        `yaml:"dev_mode"`
	TemplateFiles  map[string]jira.IssueFields `yaml:"-"`
}

type Jira struct {
	Endpoint string `yaml:"endpoint"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Config
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if c.EnableSSL {
		if c.TLSPEM.PrivateKey == "" || c.TLSPEM.CertChain == "" {
			return fmt.Errorf("Error parsing PEM blocks of router.tls_pem, missing cert or key.")
		}

		certificate, err := tls.X509KeyPair([]byte(c.TLSPEM.CertChain), []byte(c.TLSPEM.PrivateKey))
		if err != nil {
			errMsg := fmt.Sprintf("Error loading key pair: %s", err.Error())
			return fmt.Errorf(errMsg)
		}
		c.SSLCertificate = certificate
	}
	if c.Listen == "" {
		c.Listen = "0.0.0.0:9000"
	}
	if c.TemplateDir == "" {
		c.TemplateDir = "templates"
	}

	templateFilesRaw, err := filepath.Glob(filepath.Join(c.TemplateDir, "*.json"))
	if err != nil {
		return err
	}

	templateFiles := make(map[string]jira.IssueFields)
	for _, fileRaw := range templateFilesRaw {
		b, err := ioutil.ReadFile(fileRaw)
		if err != nil {
			return fmt.Errorf("error on file '%s': %s", fileRaw, err.Error())
		}
		var issueFields jira.IssueFields
		err = json.Unmarshal(b, &issueFields)
		if err != nil {
			return fmt.Errorf("error on file '%s': %s", fileRaw, err.Error())
		}
		templateFiles[strings.TrimSuffix(filepath.Base(fileRaw), filepath.Ext(fileRaw))] = issueFields
	}
	c.TemplateFiles = templateFiles
	if c.Jira.Endpoint == "" {
		return fmt.Errorf("Jira endpoint must be set")
	}
	return nil
}

type Log struct {
	Level   string `yaml:"level"`
	NoColor bool   `yaml:"no_color"`
	InJson  bool   `yaml:"in_json"`
}

func (c *Log) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Log
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	log.SetFormatter(&log.TextFormatter{
		DisableColors: c.NoColor,
	})
	if c.Level != "" {
		lvl, err := log.ParseLevel(c.Level)
		if err != nil {
			return err
		}
		log.SetLevel(lvl)
	}
	if c.InJson {
		log.SetFormatter(&log.JSONFormatter{})
	}

	return nil
}

type TLSPem struct {
	CertChain  string `yaml:"cert_chain"`
	PrivateKey string `yaml:"private_key"`
}

func (c *Config) Initialize(configYAML []byte) error {
	return yaml.Unmarshal(configYAML, &c)
}

func InitConfigFromFile(filename string) (*Config, error) {

	c := &Config{}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = c.Initialize(b)
	if err != nil {
		return nil, err
	}

	return c, nil
}
