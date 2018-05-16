//
// Copyright (c) 2018 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: EPL-2.0
//

package reify

import (
	"bytes"
	"html/template"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Reify struct{}

type ReifyInterface interface {
	Reify(templateFileName string, templateValues interface{}) (json []byte, err error)
}

// Reify returns the resulting JSON by expanding the template using the
// supplied data.
func (r *Reify) Reify(templateFileName string, templateValues interface{}) (json []byte, err error) {
	// Due to a weird quirk of go templates, we must pass the base name of the
	// template file to template.New otherwise execute can fail!
	baseFileName := filepath.Base(templateFileName)
	tmpl := template.New(baseFileName).Funcs(template.FuncMap{
		"ResourceString": func(r resource.Quantity) string {
			return (&r).String()
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
