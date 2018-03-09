package handlers

import (
	"fmt"
	kvcv1 "github.com/NervanaSystems/kube-volume-controller/pkg/apis/kvc/v1"
	"github.com/NervanaSystems/kube-volume-controller/pkg/resource"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
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
	return nil, nil
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

func (tc *testClient) Plural() string {
	return tc.plural
}

func TestS3DevHandler(t *testing.T) {

	namespace := "test"

	// Create fake clients
	fakek8sClient := fake.NewSimpleClientset()
	fakePodClient := &testClient{plural: "pods"}
	fakeNodeClient := &testClient{plural: "nodes"}
	fakePVlient := &testClient{plural: "persistentvolumes"}
	fakePVClient := &testClient{plural: "persistentvolumeclaims"}

	ownerRef := metav1.OwnerReference{}

	testCases := map[string]struct {
		volumeConfig kvcv1.VolumeConfig
		handler      DataHandler
	}{
		// S3-Dev handler
		"[s3_dev_handler] labels not set": {
			volumeConfig: kvcv1.VolumeConfig{},
			handler:      NewS3DevHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_dev_handler] awsCredentialsSecretName not set": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
			},
			handler: NewS3DevHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_dev_handler] Wrong access mode": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteMany",
			},
			handler: NewS3DevHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_dev_handler] Wrong timeoutForDataDownload format": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"timeoutForDataDownload":   "someunkownformat",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler: NewS3DevHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_dev_handler] Node List Failing": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler: NewS3DevHandler(fakek8sClient, []resource.Client{fakePodClient, &testClient{plural: "nodes", listShouldFail: true}, fakePVClient, fakePVlient}),
		},
		"[s3_dev_handler] replicas > Num nodes": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   2,
			},
			handler: NewS3DevHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_dev_handler] Any create failed": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteOnce",
				SourceURL:  "s3://foo",
			},
			handler: NewS3DevHandler(fakek8sClient, []resource.Client{&testClient{plural: "pods", createShouldFail: true}, fakeNodeClient, fakePVClient, fakePVlient}),
		},

		// S3 handler
		"[s3_handler] labels not set": {
			volumeConfig: kvcv1.VolumeConfig{},
			handler:      NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_handler] awsCredentialsSecretName not set": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
			},
			handler: NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_handler] Wrong access mode": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteMany",
			},
			handler: NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_handler] Wrong timeoutForDataDownload format": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
					"timeoutForDataDownload":   "someunkownformat",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler: NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_handler] Node List Failing": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler: NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, &testClient{plural: "nodes", listShouldFail: true}, fakePVClient, fakePVlient}),
		},
		"[s3_handler] replicas > Num nodes": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteOnce",
				Replicas:   2,
			},
			handler: NewS3Handler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[s3_handler] Any create failed": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"awsCredentialsSecretName": "foobar",
				},
				AccessMode: "ReadWriteOnce",
				SourceURL:  "s3://foo",
				Replicas:   1,
			},
			handler: NewS3Handler(fakek8sClient, []resource.Client{&testClient{plural: "pods", createShouldFail: true}, fakeNodeClient, fakePVClient, fakePVlient}),
		},

		// NFS handler
		"[nfs_handler] labels not set": {
			volumeConfig: kvcv1.VolumeConfig{},
			handler:      NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[nfs_handler] server not set": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
			},
			handler: NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[nfs_handler] path not set": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels:  map[string]string{"foo": "bar"},
				Options: map[string]string{"server": "foo"},
			},
			handler: NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[nfs_handler] Wrong access mode": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"server": "foo",
					"path":   "/",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler: NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, fakePVClient, fakePVlient}),
		},
		"[nfs_handler] Any create failed": {
			volumeConfig: kvcv1.VolumeConfig{
				Labels: map[string]string{"foo": "bar"},
				Options: map[string]string{
					"server": "foo",
					"path":   "/",
				},
				AccessMode: "ReadWriteOnce",
			},
			handler: NewNFSHandler(fakek8sClient, []resource.Client{fakePodClient, fakeNodeClient, &testClient{plural: "persistentvolumeclaims", createShouldFail: true}, fakePVlient}),
		},
	}

	for key, tc := range testCases {
		t.Logf("Testing for: %v", key)
		volume := tc.handler.OnAdd(namespace, tc.volumeConfig, ownerRef)

		// Assert stuff
		require.NotNil(t, volume)
		require.NotEqual(t, volume.Message, kvcv1.SuccessfulVolumeStatusMessage)
	}
}
