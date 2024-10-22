package main

import (
	"github.com/rs/zerolog/log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Labelstore represents an interface defining methods for connecting to a
// label store and retrieving labels associated with a given OAuth token.
type Labelstore interface {
	// Connect establishes a connection with the label store using App configuration.
	Connect(App) error
	// GetLabels retrieves labels associated with the provided OAuth token.
	// Returns a map containing the labels and a boolean indicating whether
	// the label is cluster-wide or not.
	GetLabels(token OAuthToken) (Filter, bool)
}

// WithLabelStore initializes and connects to a LabelStore specified in the
// application configuration. It assigns the connected LabelStore to the App
// instance and returns it. If the LabelStore type is unknown or an error
// occurs during the connection, it logs a fatal error.
func (a *App) WithLabelStore() *App {
	switch a.Cfg.Web.LabelStoreKind {
	case "configmap":
		a.LabelStore = &ConfigMapHandler{}
	// case "mysql":
	// 	a.LabelStore = &MySQLHandler{}
	default:
		log.Fatal().Str("type", a.Cfg.Web.LabelStoreKind).Msg("Unknown label store type")
	}
	err := a.LabelStore.Connect(*a)
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to labelstore")
	}
	return a
}

type ConfigMapHandler struct {
	labels ACLs
}

type ConfigMapACLs map[Identifier]map[LabelKey][]LabelVal

func (c ConfigMapACLs) AsMap() ACLs {
	hf := make(ACLs)

	// Convert ConfigMapACLs to ACLs
	for id, filters := range c {
		idMap := make(map[LabelKey]map[LabelVal]bool)
		for key, values := range filters {
			innerMap := make(LabelType)
			for _, v := range values {
				innerMap[v] = true // Set bool value to true
			}
			idMap[key] = innerMap
		}
		hf[id] = idMap
	}

	return hf
}

func (c *ConfigMapHandler) Connect(_ App) error {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigName("labels")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/config/labels/")
	v.AddConfigPath("./configs")
	err := v.MergeInConfig()
	if err != nil {
		return err
	}
	// var rawlabels = map[LabelKey][]LabelVal{}
	var rawlabels = ConfigMapACLs{}
	err = v.Unmarshal(&rawlabels)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while unmarshalling config file")
		return err
	}
	c.labels = rawlabels.AsMap()

	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().Str("file", e.Name).Msg("Config file changed")
		err = v.MergeInConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("Error while unmarshalling config file")
		}
		err = v.Unmarshal(&rawlabels)
		if err != nil {
			log.Fatal().Err(err).Msg("Error while unmarshalling config file")
		}
		c.labels = rawlabels.AsMap()
	})
	v.WatchConfig()
	log.Debug().Any("labels", c.labels).Msg("")
	return nil
}

func (c *ConfigMapHandler) GetLabels(token OAuthToken) (Filter, bool) {
	username := token.PreferredUsername
	groups := token.Groups
	mergedNamespaces := make(Filter)
	for label, values := range c.labels[username] {
		if _, ok := mergedNamespaces[label]; !ok {
			mergedNamespaces[label] = make(LabelType)
		}
		for value := range values {
			mergedNamespaces[label][value] = true
			if value == "#cluster-wide" {
				return nil, true
			}
		}
	}
	for _, group := range groups {
		for label, values := range c.labels[group] {
			if _, ok := mergedNamespaces[label]; !ok {
				mergedNamespaces[label] = make(LabelType)
			}
			for value := range values {
				mergedNamespaces[label][value] = true
				if value == "#cluster-wide" {
					return nil, true
				}
			}
		}
	}
	return mergedNamespaces, false
}
