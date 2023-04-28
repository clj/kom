package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type InventreePlugin struct {
	// Globalish stuff
	httpClient      *http.Client
	inventreeConfig struct {
		server   string
		userName string
		apiToken string
	}
	categoryMapping map[string]int
	// Per table stuff
	categories []int
	defaults   struct {
		symbol    string
		footprint string
	}
}

func (p *InventreePlugin) updateCategoryMapping() error {
	type category struct {
		Pk         int    `json:"pk"`
		Pathstring string `json:"pathstring"`
	}
	var categories = []category{}
	if err := p.apiGet("/api/part/category/", nil, &categories); err != nil {
		return err
	}

	p.categoryMapping = make(map[string]int)
	for _, category := range categories {
		p.categoryMapping[category.Pathstring] = category.Pk
	}
	return nil
}

func (p *InventreePlugin) updateCategories(categories []string) error {
	for _, category := range categories {
		id, ok := p.categoryMapping[category] // XXX: is it an error if the category doesn't exist?
		if !ok {
			continue
		}
		p.categories = append(p.categories, id)
	}

	return nil
}

func (p *InventreePlugin) apiGet(resource string, args map[string]string, result any) error {
	request, err := http.NewRequest("GET", p.inventreeConfig.server+resource, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Token %s", p.inventreeConfig.apiToken))
	if args != nil {
		q := request.URL.Query()
		for key, val := range args {
			q.Add(key, val)
		}
		request.URL.RawQuery = q.Encode()
	}
	response, err := p.httpClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("unexpected status code %s", response.Status)
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(result)
	if err != nil {
		return err
	}

	return nil
}

func (p *InventreePlugin) Init(api KomPluginApi, args PluginArguments) error {
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
			Token string `json:"token"`
		}
		decoder := json.NewDecoder(response.Body)
		val := &Token{}
		err = decoder.Decode(val)
		if err != nil {
			return err
		}
		p.inventreeConfig.apiToken = val.Token
		api.WriteSetting("api_token", val.Token)
	}

	api.DeleteSetting("password")

	if err = p.updateCategoryMapping(); err != nil {
		return err
	}

	categories, ok := args["categories"]
	if !ok {
		return fmt.Errorf("categories is a required argument")
	}
	if err = p.updateCategories(strings.Split(categories, ",")); err != nil {
		return err
	}

	p.defaults.symbol = args["default_symbol"]
	p.defaults.footprint = args["default_footprint"]

	return nil
}

func (p *InventreePlugin) ColumnNames() []string {
	return []string{"PK", "IPN", "Description", "Keywords", "Symbols", "Footprints"}
}

func (p *InventreePlugin) GetParts() ([]map[string]string, error) {
	type part struct {
		Pk          int    `json:"pk"`
		Ipn         string `json:"IPN"`
		Keywords    string `json:"keywords"`
		Description string `json:"description"`
	}
	args := make(map[string]string)
	args["category"] = strconv.Itoa(p.categories[0]) // XXX: disallow multiple categories, or make multiple queries
	var parts = []part{}
	if err := p.apiGet("/api/part/", args, &parts); err != nil {
		return nil, err
	}

	var result []map[string]string
	for _, part := range parts {
		partResult := make(map[string]string)
		partResult["PK"] = strconv.Itoa(part.Pk) // XXX: other types than string in here
		partResult["IPN"] = part.Ipn
		partResult["Keywords"] = part.Keywords
		partResult["Description"] = part.Description
		partResult["Symbols"] = p.defaults.symbol
		partResult["Footprints"] = p.defaults.footprint

		result = append(result, partResult)
	}

	return result, nil
}
