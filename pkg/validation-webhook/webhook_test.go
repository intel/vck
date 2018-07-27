package main

import (
	"testing"

	vckv1alpha1 "github.com/IntelAI/vck/pkg/apis/vck/v1alpha1"
	"github.com/stretchr/testify/require"
)

type testClient struct {
	plural           string
	listShouldFail   bool
	createShouldFail bool
}

func TestHandler(t *testing.T) {

	failTestCases := map[string]struct {
		volumeManager vckv1alpha1.VolumeManager
		failedMessage string
	}{
		"s3 tests missing test": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Options:    map[string]string{},
						},
					},
				},
			},
			failedMessage: "labels cannot be empty. replicas cannot be empty or less than 1. " +
				"awsCredentialsSecretName key has to be set in options. sourceURL has " +
				"to be set in options. sourceURL has to be a valid URL.",
		},
		"s3 tests incorrect values test": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							Replicas:   0,
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Options: map[string]string{
								"sourceURL":                "foo",
								"endpointURL":              "bar",
								"awsCredentialsSecretName": "secret",
							},
						},
					},
				},
			},
			failedMessage: "labels cannot be empty. replicas cannot be empty or less " +
				"than 1. sourceURL has to be a valid URL. endpointURL has to be a valid URL.",
		},
		"nfs tests": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							SourceType: "NFS",
							AccessMode: "ReadWriteMmany",
						},
					},
				},
			},
			failedMessage: "labels cannot be empty. server has to be set in options. path has to be set in options.",
		},
		"pachyderm tests": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							SourceType: "Pachyderm",
							AccessMode: "ReadWriteOnce",
						},
					},
				},
			},
			failedMessage: "labels cannot be empty. replicas cannot be empty or less than 1. " +
				"repo has to be set in options. branch has to be set in options. inputPath has to " +
				"be set in options. outputPath has to be set in options.",
		},
		"multiple id test": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							Replicas:   1,
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"sourceURL":                "s3://foo",
								"awsCredentialsSecretName": "secret",
							},
						},
						{
							ID:         "vol1",
							Replicas:   1,
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"sourceURL":                "s3://foo",
								"awsCredentialsSecretName": "secret",
							},
						},
					},
				},
			},
			failedMessage: "Cannot have duplicate id: vol1.",
		},
	}

	successTestCases := map[string]struct {
		volumeManager vckv1alpha1.VolumeManager
	}{
		"s3 tests": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							Replicas:   3,
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"sourceURL":                "s3://foo",
								"awsCredentialsSecretName": "secret",
							},
						},
					},
				},
			},
		},
		"nfs tests": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							Replicas:   1,
							SourceType: "NFS",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"server": "s3://foo",
								"path":   "secret",
							},
						},
					},
				},
			},
		},
		"pachyderm tests": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							Replicas:   1,
							SourceType: "Pachyderm",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"repo":       "foo",
								"branch":     "bar",
								"inputPath":  "/foo/",
								"outputPath": "/bar/",
							},
						},
					},
				},
			},
		},
		"multiple id test": {
			volumeManager: vckv1alpha1.VolumeManager{
				Spec: vckv1alpha1.VolumeManagerSpec{
					VolumeConfigs: []vckv1alpha1.VolumeConfig{
						{
							ID:         "vol1",
							Replicas:   1,
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"sourceURL":                "s3://foo",
								"awsCredentialsSecretName": "secret",
							},
						},
						{
							ID:         "vol2",
							Replicas:   1,
							SourceType: "S3",
							AccessMode: "ReadWriteOnce",
							Labels: map[string]string{
								"key3": "val3",
								"key4": "val4",
							},
							Options: map[string]string{
								"sourceURL":                "s3://foo",
								"awsCredentialsSecretName": "secret",
							},
						},
					},
				},
			},
		},
	}

	for key, tc := range failTestCases {
		t.Logf("Testing for: %v", key)
		response := *validateVolumeManager(tc.volumeManager)

		// Assert stuff
		require.Equal(t, false, response.Allowed)
		require.Equal(t, tc.failedMessage, response.Result.Message)
	}

	for key, tc := range successTestCases {
		t.Logf("Testing for: %v", key)
		response := *validateVolumeManager(tc.volumeManager)

		// Assert stuff
		require.Equal(t, true, response.Allowed)
	}
}
