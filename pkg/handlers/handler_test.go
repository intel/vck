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

package handlers

import (
	"fmt"
	"testing"

	vckv1alpha1 "github.com/IntelAI/vck/pkg/apis/vck/v1alpha1"

	"github.com/IntelAI/vck/pkg/resource"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type testClient struct {
	plural           string
	listShouldFail   bool
	createShouldFail bool
}

func (tc *testClient) Reify(templateValues interface{}) ([]byte, error) {
	return []byte{}, nil
}

func (tc *testClient) Create(namespace string, templateValues interface{}) error {
	if tc.createShouldFail {
		return fmt.Errorf("create failed")
	}
	return nil
}

func (tc *testClient) Delete(namespace string, name string) error {
	return nil
}

func (tc *testClient) Get(namespace, name string) (runtime.Object, error) {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{},
		},
	}, nil
}

func (tc *testClient) List(namespace string, labels map[string]string) ([]metav1.Object, error) {
	if tc.listShouldFail {
		return nil, fmt.Errorf("list failed")
	}
	return []metav1.Object{&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}}, nil
}

func (tc *testClient) Update(object runtime.Object) (runtime.Object, error) {
	return nil, nil
}

func (tc *testClient) Plural() string {
	return tc.plural
}

func TestHandler(t *testing.T) {

	namespace := "test"

	// Create fake clients
	fakek8sClient := fake.NewSimpleClientset()
	fakePodClient := &testClient{plural: "pods"}
	fakeNodeClient := &testClient{plural: "nodes"}
	fakePVlient := &testClient{plural: "persistentvolumes"}
	fakePVClient := &testClient{plural: "persistentvolumeclaims"}

	ownerRef := metav1.OwnerReference{}

	testCases := map[string]struct {
		volumeConfig  vckv1alpha1.VolumeConfig
		handler       DataHandler
		failedMessage string
	}{
		// S3 handler
		"[s3_handler] Wrong access mode": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteMany",
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "access mode has to be ReadWriteOnce",
		},
		"[s3_handler] Wrong timeoutForDataDownload format": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"timeoutForDataDownload":   "someunkownformat",
					"sourceURL":                "s3://foo",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "error while parsing timeout for data download",
		},
		"[s3_handler] Node List Failing": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"sourceURL":                "s3://foo",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, &testClient{plural: "nodes", listShouldFail: true}, fakePVClient, fakePVlient}),
			failedMessage: "error getting node list",
		},
		"[s3_handler] replicas > Num nodes": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"sourceURL":                "s3://foo",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   2,
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "replicas [2] greater than number of nodes [1]",
		},
		"[s3_handler] Invalid distribution strategy": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"sourceURL":                "s3://foo",
					"distributionStrategy":     "foo/bar",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   1,
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "invalid distributionStrategy",
		},
		"[s3_handler] # replicas in distribution strategy != # replicas": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"sourceURL":                "s3://foo",
					"distributionStrategy":     "{\"*bar*\": 2}",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   1,
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "does not match number or replicas provided",
		},
		"[s3_handler] Any create failed": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"sourceURL":                "s3://foo",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   1,
			},
			handler:       NewS3Handler(fakek8sClient, []resource.Client{&testClient{plural: "pods", createShouldFail: true}, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "error during sub-resource",
		},

		// NFS handler
		"[nfs_handler] Wrong access mode": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"server": "foo",
					"path":   "/",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler:       NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "access mode has to be either ReadWriteMany or ReadOnlyMany",
		},
		"[nfs_handler] Any create failed": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"server": "foo",
					"path":   "/",
				},
				AccessMode: "ReadWriteMany",
			},
			handler:       NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, &testClient{plural: "persistentvolumeclaims", createShouldFail: true}, fakePVlient}),
			failedMessage: "error during sub-resource",
		},

		// Pachyderm handler
		"[pachyderm_handler] Wrong access mode": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"repo":       "foo",
					"branch":     "master",
					"inputPath":  "s3/",
					"outputPath": "s3/",
				},
				AccessMode: "ReadWriteMany",
			},
			handler:       NewPachydermHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "access mode has to be ReadWriteOnce",
		},
		"[pachyderm_handler] replicas > Num nodes": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"repo":       "foo",
					"branch":     "master",
					"inputPath":  "s3/",
					"outputPath": "s3/",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   2,
			},
			handler:       NewPachydermHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "replicas [2] greater than number of nodes [1]",
		},
		"[pachyderm_handler] Any create failed": {
			volumeConfig: vckv1alpha1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"repo":       "foo",
					"branch":     "master",
					"inputPath":  "s3/",
					"outputPath": "s3/",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   1,
			},
			handler:       NewPachydermHandler(fakek8sClient, []resource.Client{&testClient{plural: "pods", createShouldFail: true}, fakeNodeClient, fakePVClient, fakePVlient}),
			failedMessage: "error during sub-resource",
		},
	}

	for key, tc := range testCases {
		t.Logf("Testing for: %v", key)
		volume := tc.handler.OnAdd(namespace, tc.volumeConfig, ownerRef)

		// Assert stuff
		require.NotNil(t, volume)
		require.NotEqual(t, volume.Message, vckv1alpha1.SuccessfulVolumeStatusMessage)
		require.Contains(t, volume.Message, tc.failedMessage)
	}
}
