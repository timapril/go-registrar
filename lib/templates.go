package lib

import (
	"errors"
	"html/template"
)

// LoadTemplates will use the configuration provided and try to load
// the templates in that directory.
func LoadTemplates(path string) *template.Template {
	// http://stackoverflow.com/questions/17206467/go-how-to-render-multiple-templates-in-golang
	// http://stackoverflow.com/questions/18276173/calling-a-template-with-several-pipeline-parameters
	templates := template.Must(template.New("").Funcs(template.FuncMap{
		"dict": dictify,
	}).ParseGlob(path))

	return templates
}

const tokensPerMapEntry = 2

func dictify(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}

	dict := make(map[string]interface{}, len(values)/tokensPerMapEntry)

	for idx := 0; idx < len(values); idx += tokensPerMapEntry {
		key, ok := values[idx].(string)

		if !ok {
			return nil, errors.New("dict keys must be strings")
		}

		dict[key] = values[idx+1]
	}

	return dict, nil
}
