package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

type fieldMapping struct {
	source       []string
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
	categories     []int
	fieldMappings  map[string]fieldMapping
	fields         []string
	usesMetadata   bool
	usesParameters bool
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

func (p *InventreePlugin) addField(name string, mapping fieldMapping) {
	p.fieldMappings[name] = mapping
	if mapping.source[0] == "metadata" {
		p.usesMetadata = true
	} else if mapping.source[0] == "parameters" {
		p.usesParameters = true
	}
	for _, v := range p.fields {
		if v == name {
			return
		}
	}

	p.fields = append(p.fields, name)
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

	p.fieldMappings = make(map[string]fieldMapping)
	p.addField("PK", fieldMapping{source: []string{"pk"}, typ: "int"})
	p.addField("IPN", fieldMapping{source: []string{"IPN"}})
	p.addField("Name", fieldMapping{source: []string{"name"}})
	p.addField("Keywords", fieldMapping{source: []string{"keywords"}})
	p.addField("Description", fieldMapping{source: []string{"description"}})
	p.addField("Symbols", fieldMapping{source: []string{"metadata", "kicad", "symbols"}, defaultValue: args["default_symbol"]})
	p.addField("Footprints", fieldMapping{source: []string{"metadata", "kicad", "footprints"}, defaultValue: args["default_footprint"]})

	fields, ok := args["fields"]
	if ok {
		parsedFields, err := parseFields(fields)
		if err != nil {
			return err
		}
		for key, mapping := range parsedFields {
			p.addField(key, mapping)
		}
	}
	return nil
}

var fieldDefRegexp = regexp.MustCompile(`^(.+?):(\((.+?)\))?(.+?)(=(\((.+?)\))?(.*?))?$`)

func parseFields(fields string) (map[string]fieldMapping, error) {
	result := make(map[string]fieldMapping)
	splitFields := strings.Split(fields, ",")

	for _, fieldDef := range splitFields {
		fieldDef = strings.TrimSpace(fieldDef)
		parsedFieldDef := fieldDefRegexp.FindStringSubmatch(fieldDef)
		if parsedFieldDef == nil {
			return nil, fmt.Errorf("could not parse field `%s`", fieldDef)
		}

		defaultValue, err := Convert(parsedFieldDef[8], parsedFieldDef[7])
		if err != nil {
			return nil, err
		}
		field := fieldMapping{
			source:       strings.Split(parsedFieldDef[4], "."),
			typ:          parsedFieldDef[3],
			defaultValue: defaultValue,
		}

		result[parsedFieldDef[1]] = field
	}

	return result, nil
}

func mangleParameters(params []map[string]any) map[string]any {
	result := make(map[string]any)
	for _, param := range params {
		name := param["template_detail"].(map[string]any)["name"].(string)
		data := param["data"].(string)

		result[name] = data
	}

	return result
}

func (p *InventreePlugin) ColumnNames() []string {
	return p.fields
}

func (p *InventreePlugin) CanFilter(column string) bool {
	return column == "PK"
}

func (p *InventreePlugin) GetParts(pkValue any) (Parts, error) {
	type part map[string]any

	var parts []part
	var partMetadata map[string]any
	var partParameters map[string]any

	if pkValue != nil {
		var part = part{}

		getPart := func() error {
			if err := p.apiGet(fmt.Sprintf("/api/part/%v/", pkValue), nil, &part); err != nil {
				return err
			}
			parts = append(parts, part)

			return nil
		}

		if p.usesMetadata || p.usesParameters {
			g := new(errgroup.Group)

			g.Go(getPart)

			if p.usesMetadata {
				g.Go(func() error {
					if err := p.apiGet(fmt.Sprintf("/api/part/%v/metadata/", pkValue), nil, &partMetadata); err != nil {
						return err
					}

					return nil
				})
			}

			if p.usesParameters {
				g.Go(func() error {
					var rawPartParameters []map[string]any
					args := make(map[string]string)
					value, _ := Convert(pkValue, "string")
					args["part"] = value.(string)

					if err := p.apiGet("/api/part/parameter/", nil, &rawPartParameters); err != nil {
						return err
					}

					partParameters = mangleParameters(rawPartParameters)

					return nil
				})
			}

			if err := g.Wait(); err != nil {
				return nil, err
			}
		} else {
			if err := getPart(); err != nil {
				return nil, err
			}
		}
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
			var value any
			var ok bool

			switch mapping.source[0] {
			case "metadata":
				value = partMetadata
				for _, key := range mapping.source {
					value, ok = value.(map[string]any)[key]
					if !ok {
						break
					}
				}
			case "parameters":
				value = partParameters
				for _, key := range mapping.source[1:] {
					value, ok = value.(map[string]any)[key]
					if !ok {
						break
					}
				}
			default:
				value, ok = part[mapping.source[0]]
			}

			if ok {
				if mapping.typ != "" {
					var err error
					value, err = Convert(value, mapping.typ)
					if err != nil {
						return nil, err
					}
				}
				partResult[field] = value
			} else {
				partResult[field] = mapping.defaultValue
			}
		}

		result = append(result, partResult)
	}

	return result, nil
}
