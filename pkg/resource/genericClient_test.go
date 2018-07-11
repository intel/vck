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

package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type fakeReify struct {
	podJson []byte
}

func (f *fakeReify) Reify(templateFileName string, templateValues interface{}) (json []byte, err error) {
	return f.podJson, nil
}

func getJSON(version, kind, name string) []byte {
	return []byte(fmt.Sprintf(`{"apiVersion": %q, "kind": %q, "metadata": {"name": %q}}`, version, kind, name))
}

func getListJSON(version, kind string, items ...[]byte) []byte {
	json := fmt.Sprintf(`{"apiVersion": %q, "kind": %q, "items": [%s]}`,
		version, kind, bytes.Join(items, []byte(",")))
	return []byte(json)
}

func getClientServer(gv *schema.GroupVersion, h func(http.ResponseWriter, *http.Request)) (dynamic.Interface, *httptest.Server, error) {
	srv := httptest.NewServer(http.HandlerFunc(h))
	cl, err := dynamic.NewClient(&restclient.Config{
		Host:          srv.URL,
		ContentConfig: restclient.ContentConfig{GroupVersion: gv},
	})
	if err != nil {
		srv.Close()
		return nil, nil, err
	}
	return cl, srv, nil
}

func compareJson(t *testing.T, json1, json2 []byte) {
	//pod1 := &corev1.Pod{}
	//pod2 := &corev1.Pod{}
	obj1 := &unstructured.Unstructured{}
	obj2 := &unstructured.Unstructured{}

	err := json.Unmarshal(json1, obj1)
	require.Nil(t, err)

	err = json.Unmarshal(json2, obj2)
	require.Nil(t, err)

	require.True(t, reflect.DeepEqual(obj1, obj2))
}

func TestGenericClient(t *testing.T) {

	namespace := "test"
	// Create scheme to convert objects
	// NOTE: If a new client is added, make sure it's added in AddKnownTypes, or if another gvk is used, a new scheme is to be used and created
	// At which point this should also be included in the test case.
	corev1Scheme := runtime.NewScheme()
	corev1Scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.PersistentVolume{}, &corev1.Pod{}, &corev1.Node{}, &corev1.PersistentVolumeClaim{})

	testCases := map[string]struct {
		apiResource  *metav1.APIResource
		resourceName string
	}{
		"pod": {
			apiResource: &metav1.APIResource{
				Kind:       "Pod",
				Name:       "pods",
				Group:      "",
				Version:    "v1",
				Namespaced: true,
			},
			resourceName: "pod1",
		},
		"node": {
			apiResource: &metav1.APIResource{
				Kind:       "Node",
				Name:       "nodes",
				Group:      "",
				Version:    "v1",
				Namespaced: false,
			},
			resourceName: "node1",
		},
		"pv": {
			apiResource: &metav1.APIResource{
				Kind:       "PersistentVolume",
				Name:       "persistentvolumes",
				Group:      "",
				Version:    "v1",
				Namespaced: false,
			},
			resourceName: "pv1",
		},
		"pvc": {
			apiResource: &metav1.APIResource{
				Kind:       "PersistentVolumeClaim",
				Name:       "persistentvolumeclaims",
				Group:      "",
				Version:    "v1",
				Namespaced: true,
			},
			resourceName: "pvc1",
		},
	}

	for key, test := range testCases {
		t.Logf("Testing for %v", key)
		resourceJson := getJSON(test.apiResource.Version, test.apiResource.Kind, test.resourceName)
		resourceList := getListJSON(test.apiResource.Version, test.apiResource.Kind, resourceJson)
		// Test Get
		client, server, err := getClientServer(&corev1.SchemeGroupVersion, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", runtime.ContentTypeJSON)
			switch r.Method {
			case "GET":
				if strings.Contains(r.URL.Path, test.resourceName) {
					// Indicates a get
					w.Write(resourceJson)
				} else {
					// Indicates a list
					w.Write(resourceList)
				}

			case "POST":
				data, err := ioutil.ReadAll(r.Body)
				require.Nil(t, err)
				require.NotNil(t, data)

				compareJson(t, data, resourceJson)
				w.Write(data)
			case "Delete":
				require.True(t, strings.Contains(r.URL.Path, test.resourceName))
				statusOK := &metav1.Status{
					TypeMeta: metav1.TypeMeta{Kind: "Status"},
					Status:   metav1.StatusSuccess,
				}
				unstructured.UnstructuredJSONScheme.Encode(statusOK, w)
			}
		})
		require.NotNil(t, client)
		require.NotNil(t, server)
		require.Nil(t, err)

		// Get resource
		resourceClient := client.Resource(test.apiResource, namespace)

		// Create the generic client
		genericClient := NewGenericClient(resourceClient, "", test.apiResource.Name, corev1Scheme, corev1.SchemeGroupVersion, &fakeReify{podJson: resourceJson})

		// Test Create
		err = genericClient.Create(namespace, nil)
		require.Nil(t, err)

		// Test Get
		obj, err := genericClient.Get(namespace, test.resourceName)
		require.NotNil(t, obj)
		require.Nil(t, err)

		// Test List
		list, err := genericClient.List(namespace, map[string]string{})
		require.NotNil(t, list)
		require.Nil(t, err)
		require.Equal(t, 1, len(list))
		toCompareJson, err := json.Marshal(list[0])
		require.NotNil(t, toCompareJson)
		require.Nil(t, err)
		compareJson(t, toCompareJson, resourceJson)

		// Test Delete
		err = genericClient.Delete(namespace, test.resourceName)
		require.Nil(t, err)

		// Close the server
		server.Close()
	}

}
