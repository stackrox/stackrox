package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"strings"
)

type basicAuth struct {
	username string
	password string
}

type dockerConfig struct {
	Auths map[string]authItem `json:"auths"`
}

type authItem struct {
	Auth  string `json:"auth"`
	Token string `json:"identitytoken"`
}

func readDockerConfig(file string) (map[string]*basicAuth, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var d dockerConfig
	if err := json.Unmarshal(bytes, &d); err != nil {
		return nil, err
	}
	basicAuthMap := make(map[string]*basicAuth)
	for registry, auth := range d.Auths {
		var username, password string
		sDec, err := base64.URLEncoding.DecodeString(auth.Auth)
		if err != nil {
			log.Errorf("Could not base64 decode %v", auth.Auth)
			continue
		}
		if auth.Token != "" {
			username = strings.TrimSuffix(auth.Auth, ":")
			password = auth.Token
		} else {
			spl := strings.SplitN(string(sDec), ":", 2)
			if len(spl) != 2 {
				log.Errorf("Could not parse registry auth for %v", registry)
				continue
			}
			username = strings.TrimSpace(spl[0])
			password = strings.TrimSpace(spl[1])
		}
		basicAuthMap[registry] = &basicAuth{username: username, password: password}
	}
	return basicAuthMap, nil
}
