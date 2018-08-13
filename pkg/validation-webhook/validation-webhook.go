package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	vckv1alpha1 "github.com/IntelAI/vck/pkg/apis/vck/v1alpha1"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

//Config struct
type Config struct {
	CertFile string
	KeyFile  string
}

var scheme = runtime.NewScheme()

var codecs = serializer.NewCodecFactory(scheme)

type admitFunc func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func toAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func admitVolumeManager(ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.V(2).Info("admitting volume manager")

	raw := ar.Request.Object.Raw
	vm := vckv1alpha1.VolumeManager{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &vm); err != nil {
		glog.Error(err)
		return toAdmissionResponse(err)
	}

	reviewResponse := validateVolumeManager(vm)

	return reviewResponse
}

func validateNFS(vc vckv1alpha1.VolumeConfig) string {
	errs := []string{}
	if len(vc.Labels) == 0 {
		errs = append(errs, "labels cannot be empty.")
	}

	if server, ok := vc.Options["server"]; !ok {
		errs = append(errs, "server has to be set in options.")
	} else {
		if ip := net.ParseIP(server); ip == nil {
			errs = append(errs, "server is not a valid IP.")
		}
	}

	if _, ok := vc.Options["path"]; !ok {
		errs = append(errs, "path has to be set in options.")
	}

	if vc.AccessMode == "" {
		errs = append(errs, "accessMode has to be set.")
	}

	if vc.AccessMode != "ReadWriteMany" && vc.AccessMode != "ReadOnlyMany" {
		if vc.AccessMode == "" {
			errs = append(errs, "accessMode has to be set.")
		} else {
			errs = append(errs, "accessMode must be ReadWriteMany or ReadOnlyMany.")
		}
	}

	return strings.Join(errs, " ")
}

func validateS3(vc vckv1alpha1.VolumeConfig) string {
	errs := []string{}
	if len(vc.Labels) == 0 {
		errs = append(errs, "labels cannot be empty.")
	}

	if vc.Replicas < 1 {
		errs = append(errs, "replicas cannot be empty or less than 1.")
	}

	if _, ok := vc.Options["awsCredentialsSecretName"]; !ok {
		errs = append(errs, "awsCredentialsSecretName key has to be set in options.")
	}

	if _, ok := vc.Options["sourceURL"]; !ok {
		errs = append(errs, "sourceURL has to be set in options.")
	}

	if _, err := url.ParseRequestURI(vc.Options["sourceURL"]); err != nil {
		errs = append(errs, "sourceURL has to be a valid URL.")
	}

	if endpointURL, ok := vc.Options["endpointURL"]; ok {
		if _, err := url.ParseRequestURI(endpointURL); err != nil {
			errs = append(errs, "endpointURL has to be a valid URL.")
		}
	}

	if timeoutForDataDownload, ok := vc.Options["timeoutForDataDownload"]; ok {
		if _, err := time.ParseDuration(timeoutForDataDownload); err != nil {
			errs = append(errs, "timeoutForDataDownload has the incorrect format.")
		}
	}

	if vc.AccessMode != "ReadWriteOnce" {
		if vc.AccessMode == "" {
			errs = append(errs, "accessMode has to be set.")
		} else {
			errs = append(errs, "accessMode must be ReadWriteOnce.")
		}
	}

	return strings.Join(errs, " ")
}

func validatePachyderm(vc vckv1alpha1.VolumeConfig) string {
	errs := []string{}
	if len(vc.Labels) == 0 {
		errs = append(errs, "labels cannot be empty.")
	}

	if vc.Replicas < 1 {
		errs = append(errs, "replicas cannot be empty or less than 1.")
	}

	if _, ok := vc.Options["repo"]; !ok {
		errs = append(errs, "repo has to be set in options.")
	}

	if _, ok := vc.Options["branch"]; !ok {
		errs = append(errs, "branch has to be set in options.")
	}

	if _, ok := vc.Options["inputPath"]; !ok {
		errs = append(errs, "inputPath has to be set in options.")
	}

	if _, ok := vc.Options["outputPath"]; !ok {
		errs = append(errs, "outputPath has to be set in options.")
	}

	if timeoutForDataDownload, ok := vc.Options["timeoutForDataDownload"]; ok {
		if _, err := time.ParseDuration(timeoutForDataDownload); err != nil {
			errs = append(errs, "timeoutForDataDownload has the incorrect format.")
		}
	}

	if vc.AccessMode != "ReadWriteOnce" {
		if vc.AccessMode == "" {
			errs = append(errs, "accessMode has to be set.")
		} else {
			errs = append(errs, "accessMode must be ReadWriteOnce.")
		}
	}

	return strings.Join(errs, " ")
}

func validateVolumeManager(vm vckv1alpha1.VolumeManager) *v1beta1.AdmissionResponse {
	glog.V(2).Info("Validating Volume Manager...")
	errs := []string{}
	ids := make(map[string]bool)
	for _, vc := range vm.Spec.VolumeConfigs {
		if _, ok := ids[vc.ID]; ok {
			errs = append(errs, "Cannot have duplicate id: "+vc.ID+".")
		}
		ids[vc.ID] = true
		switch vc.SourceType {
		case "NFS":
			if err := validateNFS(vc); err != "" {
				errs = append(errs, err)
			}
		case "S3":
			if err := validateS3(vc); err != "" {
				errs = append(errs, err)
			}
		case "Pachyderm":
			if err := validatePachyderm(vc); err != "" {
				errs = append(errs, err)
			}
		}
	}

	if err := "" + strings.Join(errs, " "); err != "" {
		return &v1beta1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: err,
			},
		}
	}
	glog.V(2).Info("All Volume Manager(s) look good!")
	return &v1beta1.AdmissionResponse{
		Allowed: true,
	}
}

func serve(w http.ResponseWriter, r *http.Request, admit admitFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}
	var reviewResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Error(err)
		reviewResponse = toAdmissionResponse(err)
	} else {
		reviewResponse = admit(ar)
	}

	response := v1beta1.AdmissionReview{}
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = ar.Request.UID
	}

	// reset the Object and OldObject, they are not needed in a response.
	ar.Request.Object = runtime.RawExtension{}
	ar.Request.OldObject = runtime.RawExtension{}

	resp, err := json.Marshal(response)
	if err != nil {
		glog.Error(err)
	}
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
	}
}

func serveVolumeManager(w http.ResponseWriter, r *http.Request) {
	serve(w, r, admitVolumeManager)
}

//Main starts server
func main() {
	var config Config

	flag.StringVar(&config.CertFile, "tls-cert-file", "/etc/webhook/certs/cert.pem", ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert).")

	flag.StringVar(&config.KeyFile, "tls-private-key-file", "/etc/webhook/certs/key.pem", ""+
		"File containing the default x509 private key matching --tls-cert-file.")

	scheme.AddKnownTypes(vckv1alpha1.SchemeGroupVersion,
		&vckv1alpha1.VolumeManager{},
		&vckv1alpha1.VolumeManagerList{},
	)

	pair, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		glog.Fatalf("Failed to load key pair: %v", err)
	}

	glog.V(2).Info("Starting Server...")
	http.HandleFunc("/validate", serveVolumeManager)
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}
	glog.Fatal(server.ListenAndServeTLS("", ""))
}
