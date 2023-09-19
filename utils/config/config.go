package config

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/alecthomas/errors"
	"github.com/alecthomas/kong"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func CreateLoader(r io.Reader) (kong.Resolver, error) {
	config := map[string]interface{}{}
	err := json.NewDecoder(r).Decode(&config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode config")
	}
	flattened := flatten(config)
	return kong.ResolverFunc(func(context *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		key := ""
		if parent.Command != nil {
			key = parent.Command.Path() + "-"
		}
		key += flag.Name
		key = camelCase(key)
		return flattened[key], nil
	}), nil
}

func flatten(config map[string]interface{}) map[string]interface{} {
	caser := cases.Title(language.English)
	flat := map[string]interface{}{}
	for k, v := range config {
		switch v := v.(type) {
		case map[string]interface{}:
			for k2, v2 := range flatten(v) {
				flat[camelCase(k)+caser.String(camelCase(k2))] = v2
			}
		default:
			flat[k] = v
		}
	}
	return flat
}

func camelCase(s string) string {
	caser := cases.Title(language.English)
	out := strings.ReplaceAll(caser.String(strings.ReplaceAll(s, "-", " ")), " ", "")
	return strings.ToLower(out[:1]) + out[1:]
}
