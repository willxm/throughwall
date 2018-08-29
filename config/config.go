package config

import (
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

type Cfg struct {
	Password string
}

func Config() (*Cfg, error) {

	c := Cfg{}

	data, err := ioutil.ReadFile("../config.YAML")

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &c)

	if err != nil {
		return nil, err
	}

	return &c, nil

}
