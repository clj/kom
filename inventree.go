package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type InventreePlugin struct {
	httpClient      *http.Client
	inventreeConfig struct {
		server   string
		userName string
		apiToken string
	}
}

func (p *InventreePlugin) Init(api KomPluginApi) error {
	var err error

	p.httpClient = &http.Client{}

	p.inventreeConfig.server, err = api.ReadSetting("server")
	if err != nil {
		return err
	}
	p.inventreeConfig.userName, err = api.ReadSetting("username")
	if err != nil {
		return err
	}
	p.inventreeConfig.apiToken, err = api.ReadSetting("api_token")
	if err != nil {
		var password string

		password, err = api.ReadSetting("password")
		if err != nil {
			return err
		}
		request, err := http.NewRequest("GET", p.inventreeConfig.server+"/api/user/token", nil)
		if err != nil {
			return err
		}
		request.SetBasicAuth(p.inventreeConfig.userName, password)
		response, err := p.httpClient.Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode != 200 {
			return fmt.Errorf("unexpected status code %s", response.Status)
		}

		type Token struct {
			Token string `json:"token,omitempty"`
		}
		decoder := json.NewDecoder(response.Body)
		val := &Token{}
		err = decoder.Decode(val)
		if err != nil {
			return err
		}
		p.inventreeConfig.apiToken = val.Token
		api.WriteSetting("api_token", val.Token)
		api.DeleteSetting("password")
	}

	return nil
}
