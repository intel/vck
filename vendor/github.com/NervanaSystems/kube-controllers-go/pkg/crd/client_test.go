package crd

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/NervanaSystems/kube-controllers-go/pkg/states"
	"github.com/stretchr/testify/require"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apimachinery"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
)

const (
	testCRDJSON = `{"kind":"TestCRD","apiVersion":"test.intel.com/v1","metadata":{"name":"foobar","namespace":"test-intel","creationTimestamp":null}}
`
)

var (
	testCRD = &TestCRD{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestCRD",
			APIVersion: "test.intel.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foobar",
			Namespace: "test-intel",
		},
	}

	testHandle = New(
		&TestCRD{},
		&TestCRDList{},
		"test.intel.com",
		"v1",
		"TestCRD",
		"testCRD",
		"testCRDs",
		extv1beta1.NamespaceScoped,
		"file:///go/src/github.com/NervanaSystems/kube-controllers-go/resources/test_schemas/testCRD.json")
)

// Test CRD, which has nothing but apiversion, kind and metadata.
// If any fields are added to this, the boiler plate functions at the end of this file
// needs to be updated.
type TestCRD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
}

// A list of Test CRDs.
type TestCRDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []TestCRD `json:"items"`
}

func TestNewClient(t *testing.T) {
	require.Equal(t, "testCRDs.test.intel.com", testHandle.resourceName())

	_, err := NewClient(rest.Config{}, testHandle)
	require.Nil(t, err)
}

func TestCreateOK(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "POST", request.Method)

		require.Equal(t,
			testCRDJSON,
			readBody(t, request.Body))

		return httpStatus(200, "200 OK", ""), nil
	})

	err := client.Create(testCRD)
	require.Nil(t, err)
}

func TestCreateSchemaFail(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Fail(t, "Request should not make it to the API server")
		return nil, nil
	})

	// Missing type metadata (kind and api version).
	err := client.Create(&TestCRD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foobar",
			Namespace: "test-intel",
		},
	})
	require.Contains(t, err.Error(), "Invalid JSON")
}

func TestCreateFail(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		return httpStatus(500, "500 Server failure", ""), nil
	})

	err := client.Create(testCRD)
	require.NotNil(t, err)
	statusError, ok := err.(*k8serror.StatusError)
	require.True(t, ok)
	require.EqualValues(t, statusError.ErrStatus.Reason, "InternalError")
}

func TestGetOK(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "GET", request.Method)
		return httpStatus(200, "200 OK", testCRDJSON), nil
	})

	crd, err := client.Get("test-intel", "foobar")
	require.Nil(t, err)

	b, ok := crd.(*TestCRD)
	require.True(t, ok)
	require.Equal(t, b.Name(), "foobar")
	require.Equal(t, b.Namespace(), "test-intel")
}

func TestGetNotFound(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "GET", request.Method)
		return httpStatus(404, "404 Not Found", ""), nil
	})

	_, err := client.Get("test-intel", "foobar")
	require.NotNil(t, err)
	statusError, ok := err.(*k8serror.StatusError)
	require.True(t, ok)
	require.EqualValues(t, statusError.ErrStatus.Reason, "NotFound")
}

func TestGetFail(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "GET", request.Method)
		return httpStatus(500, "500 Server failure", ""), nil
	})

	_, err := client.Get("test-intel", "foobar")
	require.NotNil(t, err)
	statusError, ok := err.(*k8serror.StatusError)
	require.True(t, ok)
	require.EqualValues(t, statusError.ErrStatus.Reason, "InternalError")
}

func TestUpdateOK(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "PUT", request.Method)
		require.Equal(t, testCRDJSON, readBody(t, request.Body))

		return httpStatus(200, "200 OK", testCRDJSON), nil
	})

	crd, err := client.Update(testCRD)
	require.Nil(t, err)

	b, ok := crd.(*TestCRD)
	require.True(t, ok)
	require.Equal(t, b.Name(), "foobar")
	require.Equal(t, b.Namespace(), "test-intel")
}

func TestUpdateSchemaFail(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Fail(t, "Request should not make it to the API server")
		return nil, nil
	})

	// Missing type metadata (kind and api version).
	_, err := client.Update(&TestCRD{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foobar",
			Namespace: "test-intel",
		},
	})
	require.Contains(t, err.Error(), "Invalid JSON")
}

func TestUpdateNotFound(t *testing.T) {
	client := fakeClient(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, "PUT", request.Method)
		require.Equal(t, testCRDJSON, readBody(t, request.Body))

		return httpStatus(404, "404 Not Found", testCRDJSON), nil
	})

	_, err := client.Update(testCRD)
	require.NotNil(t, err)
}

// Reads from reader and returns string.
func readBody(t *testing.T, reader io.Reader) string {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed reading HTTP body")
	}

	return string(data)
}

// Helper to provide a response from the fakeClient handler to the client libraries.
func httpStatus(code int, status string, body string) *http.Response {
	return &http.Response{
		StatusCode: code,
		Status:     status,
		Body:       FakeBody{strings.NewReader(body)},
	}
}

// fakeClient provides a client which mimics the request/response cycle between client and the api-server, through
// a user provided function (handler).
func fakeClient(handler func(request *http.Request) (*http.Response, error)) *client {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(testHandle.SchemaGroupVersion, testHandle.ResourceType, testHandle.ResourceListType)

	// NOTE: By adding an empty group, legacy API support gets added to the rest client.
	// Some requests invoke this API and we have to add support.
	apiRegistry, _ := registered.NewAPIRegistrationManager("")
	apiRegistry.RegisterGroup(apimachinery.GroupMeta{
		GroupVersion: schema.GroupVersion{},
	})

	apiRegistry.RegisterGroup(apimachinery.GroupMeta{
		GroupVersion: schema.GroupVersion{Group: "test.intel.com", Version: "v1"},
	})

	return &client{
		&fake.RESTClient{
			GroupName:            "test.intel.com",
			VersionedAPIPath:     "/apis",
			Client:               fake.CreateHTTPClient(handler),
			NegotiatedSerializer: serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)},
			APIRegistry:          apiRegistry,
		},
		testHandle,
	}
}

// A fake body creates a ReadCloser structure from a string.
// The structure holds a reader object, which is used to forward the cursor in Read() in the
// backing string, until EOL.
type FakeBody struct {
	reader io.Reader
}

func (fr FakeBody) Read(p []byte) (n int, err error) { return fr.reader.Read(p) }
func (fr FakeBody) Close() error                     { return nil }

// Below is the boiler plating to make the TestCRD implement the DeepCopy methods, which
// in turn is needed to treat the TestCRD as a runtime.Object.
func (e *TestCRD) Name() string {
	return e.ObjectMeta.Name
}

func (e *TestCRD) Namespace() string {
	return e.ObjectMeta.Namespace
}

func (e *TestCRD) JSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (e *TestCRD) GetStatusState() states.State {
	return states.Pending
}

func (e *TestCRD) GetSpecState() states.State {
	return states.Pending
}

func (e *TestCRD) SetStatusStateWithMessage(state states.State, msg string) {}

func (in *TestCRD) DeepCopyInto(out *TestCRD) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	return
}

func (in *TestCRD) DeepCopy() *TestCRD {
	if in == nil {
		return nil
	}
	out := new(TestCRD)
	in.DeepCopyInto(out)
	return out
}

func (in *TestCRD) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *TestCRDList) DeepCopyInto(out *TestCRDList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]TestCRD, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

func (in *TestCRDList) DeepCopy() *TestCRDList {
	if in == nil {
		return nil
	}
	out := new(TestCRDList)
	in.DeepCopyInto(out)
	return out
}

func (in *TestCRDList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
