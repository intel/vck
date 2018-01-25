package reify

import (
	"bytes"
	"html/template"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Reify returns the resulting JSON by expanding the template using the
// supplied data.
func Reify(templateFileName string, templateValues interface{}, globalTemplateValues map[string]string) (json []byte, err error) {
	// Due to a weird quirk of go templates, we must pass the base name of the
	// template file to template.New otherwise execute can fail!
	baseFileName := filepath.Base(templateFileName)
	tmpl := template.New(baseFileName).Funcs(template.FuncMap{
		"ResourceString": func(r resource.Quantity) string {
			return (&r).String()
		},
		"GlobalTemplateValue": func(key string) string {
			return globalTemplateValues[key]
		},
	})
	tmpl, err = tmpl.ParseFiles(templateFileName)
	if err != nil {
		glog.Warningf("[reify] error parsing template file: %v", err)
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateValues)
	if err != nil {
		return nil, err
	}

	// Translate YAML to JSON.
	json, err = yaml.YAMLToJSON(buf.Bytes())

	glog.Infof("reified template [%s] with data [%v]:\nYAML:\n%s\n\nJSON:\n%s", templateFileName, templateValues, buf.String(), string(json))

	return
}
