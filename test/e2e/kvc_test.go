package e2e

import (
	"flag"
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/stretchr/testify/require"

	crv1 "github.com/kubeflow/experimental-kvc/pkg/apis/kvc/v1"
	crv1_client "github.com/kubeflow/experimental-kvc/pkg/client/clientset/versioned"
	crv1_volume_manager "github.com/kubeflow/experimental-kvc/pkg/client/clientset/versioned/typed/kvc/v1"
	"github.com/kubeflow/experimental-kvc/pkg/states"
	"github.com/kubeflow/experimental-kvc/pkg/util"
)

var (
	namespace   = flag.String("namespace", "e2e-test", "namespace used for the e2e test")
	s3ServerIP  = flag.String("s3serverip", "", "S3 server IP address")
	nfsServerIP = flag.String("nfsserverip", "", "NFS server IP address")
)

func makeClients(t *testing.T) (crv1_volume_manager.VolumeManagerInterface, *kubernetes.Clientset) {
	user, err := user.Current()
	require.Nil(t, err)

	config, err := util.BuildConfig(filepath.Join(user.HomeDir, ".kube/config"))
	require.Nil(t, err)

	k8sClient, err := kubernetes.NewForConfig(config)
	require.Nil(t, err)
	require.NotNil(t, k8sClient)

	crdClient, err := crv1_client.NewForConfig(config)
	require.Nil(t, err)
	require.NotNil(t, crdClient)

	return crdClient.KvcV1().VolumeManagers(*namespace), k8sClient
}

func makeVolumeManager(volumeConfigs []crv1.VolumeConfig) *crv1.VolumeManager {
	name := fmt.Sprintf("kvc-e2e-test-%s", uuid.NewUUID())
	return &crv1.VolumeManager{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: crv1.VolumeManagerSpec{
			VolumeConfigs: volumeConfigs,
			State:         states.Running,
		},
	}
}

// WaitForCRState polls for an expected CR state untill it reaches a timeout.
func waitForCRState(crdClient crv1_volume_manager.VolumeManagerInterface, name string, state states.State) error {
	return waitPoll(func() (bool, error) {
		volman, err := crdClient.Get(name, metav1.GetOptions{})
		if err == nil && volman.Status.State == state {
			return true, nil
		}

		return false, err
	})
}

func waitPoll(waitFunc func() (bool, error)) error {
	return wait.Poll(1*time.Second, 30*time.Second, waitFunc)
}

func TestVolumeManager(t *testing.T) {
	crdClient, _ := makeClients(t)

	testCases := []struct {
		description   string
		volumeConfigs []crv1.VolumeConfig
		expSuccess    bool
		expError      string
		expHP         bool
		expNA         bool
		expPVC        bool
	}{
		// Positive test cases.
		{
			description: "single vc - S3 - no error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://e2e-test/cifar-100-python.tar.gz",
					EndpointURL: fmt.Sprintf("http://%s:9000", *s3ServerIP),
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"awsCredentialsSecretName": "s3-creds",
					},
				},
			},
			expSuccess: true,
			expError:   "",
			expHP:      true,
			expNA:      true,
			expPVC:     false,
		},
		{
			description: "single vc - NFS - no error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					SourceType: "NFS",
					AccessMode: "ReadWriteMany",
					Capacity:   ".5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"server": *nfsServerIP,
						"path":   "/",
					},
				},
			},
			expSuccess: true,
			expError:   "",
			expHP:      false,
			expNA:      false,
			expPVC:     true,
		},
		{
			description: "multiple vc - S3 and NFS - no error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://e2e-test/cifar-100-python.tar.gz",
					EndpointURL: fmt.Sprintf("http://%s:9000", *s3ServerIP),
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"awsCredentialsSecretName": "s3-creds",
					},
				},
				{
					ID:         "vol1",
					SourceType: "NFS",
					AccessMode: "ReadWriteMany",
					Capacity:   ".5Gi",
					Labels: map[string]string{
						"key3": "val3",
						"key4": "val4",
					},
					Options: map[string]string{
						"server": *nfsServerIP,
						"path":   "/",
					},
				},
			},
			expSuccess: true,
			expError:   "",
			expHP:      true,
			expNA:      true,
			expPVC:     true,
		},
		{
			description: "single vc - Pachyderm - non-recursive - no error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					Replicas:   1,
					SourceType: "Pachyderm",
					AccessMode: "ReadWriteOnce",
					Capacity:   "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"repo":       "test",
						"branch":     "master",
						"inputPath":  "s3/test",
						"outputPath": "test",
					},
				},
			},
			expSuccess: true,
			expError:   "",
			expHP:      true,
			expNA:      true,
			expPVC:     false,
		},
		{
			description: "single vc - Pachyderm - recursive - no error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					Replicas:   1,
					SourceType: "Pachyderm",
					AccessMode: "ReadWriteOnce",
					Capacity:   "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"repo":       "test",
						"branch":     "master",
						"inputPath":  "s3/",
						"outputPath": "test",
					},
				},
			},
			expSuccess: true,
			expError:   "",
			expHP:      true,
			expNA:      true,
			expPVC:     false,
		},
		// Negative test cases.
		{
			description: "single vc - S3 - no label error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://e2e-test/cifar-100-python.tar.gz",
					EndpointURL: fmt.Sprintf("http://%s:9000", *s3ServerIP),
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels:      map[string]string{},
					Options: map[string]string{
						"awsCredentialsSecretName": "s3-creds",
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("labels cannot be empty"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - S3 - no creds error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://e2e-test/cifar-100-python.tar.gz",
					EndpointURL: fmt.Sprintf("http://%s:9000", *s3ServerIP),
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("awsCredentialsSecretName key has to be set in options"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - NFS - no label error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					SourceType: "NFS",
					AccessMode: "ReadWriteMany",
					Capacity:   ".5Gi",
					Labels:     map[string]string{},
					Options: map[string]string{
						"server": *nfsServerIP,
						"path":   "/",
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("labels cannot be empty"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - NFS - no server in options error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					SourceType: "NFS",
					AccessMode: "ReadWriteMany",
					Capacity:   ".5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"path": "/",
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("server has to be set in options"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - NFS - no path in options error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					SourceType: "NFS",
					AccessMode: "ReadWriteMany",
					Capacity:   ".5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"server": *nfsServerIP,
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("path has to be set in options"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - S3 - time out error due to bad url",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://fake-url",
					EndpointURL: fmt.Sprintf("http://%s:9000", *s3ServerIP),
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"awsCredentialsSecretName": "s3-creds",
						"timeoutForDataDownload":   "10s",
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("mc: <ERROR> Unable to validate source"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - S3 - timeout error due to bad endpoint",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://e2e-test/cifar-100-python.tar.gz",
					EndpointURL: "fake.end.point",
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"awsCredentialsSecretName": "s3-creds",
						"timeoutForDataDownload":   "10s",
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("mc: <ERROR> Unable to validate source"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "multiple vc - S3 and NFS - S3 failed due to no creds error",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:          "vol1",
					Replicas:    1,
					SourceType:  "S3",
					SourceURL:   "s3://e2e-test/cifar-100-python.tar.gz",
					EndpointURL: fmt.Sprintf("http://%s:9000", *s3ServerIP),
					AccessMode:  "ReadWriteOnce",
					Capacity:    "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{},
				},
				{
					ID:         "vol1",
					SourceType: "NFS",
					AccessMode: "ReadWriteMany",
					Capacity:   ".5Gi",
					Labels: map[string]string{
						"key3": "val3",
						"key4": "val4",
					},
					Options: map[string]string{
						"server": *nfsServerIP,
						"path":   "/",
					},
				},
			},
			expSuccess: false,
			expError:   fmt.Sprintf("awsCredentialsSecretName key has to be set in options"),
			expHP:      false,
			expNA:      false,
			expPVC:     false,
		},
		{
			description: "single vc - Pachyderm - ",
			volumeConfigs: []crv1.VolumeConfig{
				{
					ID:         "vol1",
					Replicas:   1,
					SourceType: "Pachyderm",
					AccessMode: "ReadWriteOnce",
					Capacity:   "5Gi",
					Labels: map[string]string{
						"key1": "val1",
						"key2": "val2",
					},
					Options: map[string]string{
						"repo":                   "test",
						"branch":                 "master",
						"inputPath":              "s3/",
						"outputPath":             "test",
						"timeoutForDataDownload": "10s",
					},
				},
			},
			expSuccess: true,
			expError:   "",
			expHP:      true,
			expNA:      true,
			expPVC:     false,
		},
	}

	for _, testCase := range testCases {
        fmt.Printf("%v n", testCase.description)
		volman := makeVolumeManager(testCase.volumeConfigs)
		createdVolman, err := crdClient.Create(volman)
		require.Nil(t, err)
		/*
			defer func() {
				delOpts := &metav1.DeleteOptions{}
				crdClient.Delete(volman.GetName(), delOpts)
			}()
		*/
		if testCase.expSuccess {
			err := waitForCRState(crdClient, createdVolman.GetName(), states.Running)
			require.Nil(t, err)
			volman, err := crdClient.Get(createdVolman.GetName(), metav1.GetOptions{})
			require.Nil(t, err)
			require.Equal(t, states.Running, volman.Status.State)

			for _, vol := range volman.Status.Volumes {
				require.Equal(t, crv1.SuccessfulVolumeStatusMessage, vol.Message)
			}

			if testCase.expHP {
				gotHP := false
				for _, vol := range volman.Status.Volumes {
					if vol.VolumeSource.HostPath != nil {
						gotHP = true
						break
					}
				}
				require.True(t, gotHP)

				if testCase.expNA {
					gotNA := false
					for _, vol := range volman.Status.Volumes {
						if vol.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
							gotNA = true
							break
						}
					}
					require.True(t, gotNA)
				}
			}

			// TODO(balajismaniam): check if the PV and PVC were created.
			if testCase.expPVC {
				gotPVC := false
				for _, vol := range volman.Status.Volumes {
					if vol.VolumeSource.PersistentVolumeClaim != nil {
						gotPVC = true
						break
					}
				}
				require.True(t, gotPVC)
			}
		}

		if !testCase.expSuccess {
			err := waitForCRState(crdClient, createdVolman.GetName(), states.Failed)
			require.Nil(t, err)
			volman, err := crdClient.Get(createdVolman.GetName(), metav1.GetOptions{})
			require.Nil(t, err)

			require.Equal(t, states.Failed, volman.Status.State)
			require.Equal(t, fmt.Sprintf("failed to deploy all the sub-resources"), volman.Status.Message)

			if testCase.expError != "" {
				gotMessage := false
				for _, vol := range volman.Status.Volumes {
					if strings.Contains(vol.Message, testCase.expError) {
						gotMessage = true
						break
					}
				}
				require.True(t, gotMessage)
			}
		}
	}
}
