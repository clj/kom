package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type fieldMapping struct {
	source       string
	typ          string
	defaultValue any
}

type InventreePlugin struct {
	// Globalish stuff?
	httpClient      *http.Client
	inventreeConfig struct {
		server   string
		userName string
		apiToken string
	}
	categoryMapping map[string]int
	// Per table stuff?
	categories    []int
	fieldMappings map[string]fieldMapping
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

	p.fieldMappings = map[string]fieldMapping{
		"PK":          {source: "pk"},
		"IPN":         {source: "IPN"},
		"Name":        {source: "name"},
		"Keywords":    {source: "keywords"},
		"Description": {source: "description"},
		"Symbols":     {source: "symbols", defaultValue: args["default_symbol"]},
		"Footprints":  {source: "footprints", defaultValue: args["default_footprint"]},
	}
	return nil
}

func (p *InventreePlugin) ColumnNames() []string {
	return []string{"PK", "IPN", "Name", "Description", "Keywords", "Symbols", "Footprints"}
}

func (p *InventreePlugin) CanFilter(column string) bool {
	return column == "PK"
}

func (p *InventreePlugin) GetParts(pkValue any) (Parts, error) {
	type part map[string]any
	var parts []part

	if pkValue != nil {
		var part = part{}

		if err := p.apiGet(fmt.Sprintf("/api/part/%v/", pkValue), nil, &part); err != nil {
			return nil, err
		}
		parts = append(parts, part)
	} else {
		args := make(map[string]string)
		args["category"] = strconv.Itoa(p.categories[0]) // XXX: possible to filter multiple at the same time? or disallow multiple categories, or make multiple queries

		if err := p.apiGet("/api/part/", args, &parts); err != nil {
			return nil, err
		}
	}

	var result Parts
	for _, part := range parts {
		partResult := make(Part)
		for field, mapping := range p.fieldMappings {
			value, ok := part[mapping.source]
			if ok {
				partResult[field] = value
			} else {
				partResult[field] = mapping.defaultValue
			}
		}

		result = append(result, partResult)
	}

	return result, nil
}
