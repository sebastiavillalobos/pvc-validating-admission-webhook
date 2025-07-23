package main

import (
    "crypto/tls"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"

    "golang.org/x/exp/slog"
    admissionv1 "k8s.io/api/admission/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
    port    int
    tlsKey  string
    tlsCert string
    maxPVCSize int64 = 5 * 1024 * 1024 * 1024 // 5 GiB
)

type PatchOperation struct {
    Op    string      `json:"op"`
    Path  string      `json:"path"`
    Value interface{} `json:"value,omitempty"`
}

// http error handling func
func httpError(w http.ResponseWriter, err error) {
    slog.Error("unable to complete request", "error", err.Error())
    w.WriteHeader(http.StatusBadRequest)
    w.Write([]byte(err.Error()))
}

// Parse Admission Review from requests
func parseAdmissionReview(req *http.Request, deserializer runtime.Decoder) (*admissionv1.AdmissionReview, error) {
    reqData, err := io.ReadAll(req.Body)
    if err != nil {
        slog.Error("error reading request body", err)
        return nil, err
    }

    admissionReviewRequest := &admissionv1.AdmissionReview{}
    _, _, err = deserializer.Decode(reqData, nil, admissionReviewRequest)
    if err != nil {
        slog.Error("unable to deserialize request", err)
        return nil, err
    }
    return admissionReviewRequest, nil
}

// validation handler 
func validate(w http.ResponseWriter, r *http.Request) {
    slog.Info("received new validation request")

    scheme := runtime.NewScheme()
    codecFactory := serializer.NewCodecFactory(scheme)
    deserializer := codecFactory.UniversalDeserializer()

    admissionReviewRequest, err := parseAdmissionReview(r, deserializer)
    if err != nil {
        httpError(w, err)
        return
    }

    pvcGVR := metav1.GroupVersionResource{
        Group:    "",
        Version:  "v1",
        Resource: "persistentvolumeclaims",
    }

    if admissionReviewRequest.Request.Resource != pvcGVR {
        err := errors.New("admission request is not of kind: PersistentVolumeClaim")
        httpError(w, err)
        return
    }

    pvc := corev1.PersistentVolumeClaim{}
    _, _, err = deserializer.Decode(admissionReviewRequest.Request.Object.Raw, nil, &pvc)
    if err != nil {
        err := errors.New("unable to unmarshal request to PersistentVolumeClaim")
        httpError(w, err)
        return
    }


    if pvc.Spec.Resources.Requests[corev1.ResourceStorage].CmpInt64(maxPVCSize) == 1 {
        err := fmt.Errorf("PVC size exceeds the maximum limit of %d bytes", maxPVCSize)
        httpError(w, err)
        return
    }

    admissionResponse := &admissionv1.AdmissionResponse{
        Allowed: true,
    }

    var admissionReviewResponse admissionv1.AdmissionReview
    admissionReviewResponse.Response = admissionResponse
    admissionReviewResponse.SetGroupVersionKind(admissionReviewRequest.GroupVersionKind())
    admissionReviewResponse.Response.UID = admissionReviewRequest.Request.UID

    responseBytes, err := json.Marshal(admissionReviewResponse)
    if err != nil {
        err := errors.New("unable to marshal patch response into bytes")
        httpError(w, err)
        return
    }
    slog.Info("validation complete", "PVC validated", pvc.ObjectMeta.Name)
    w.Write(responseBytes)
}

func main() {
    flag.IntVar(&port, "port", 9093, "Admission controller port")
    flag.StringVar(&tlsKey, "tls-key", "/etc/webhook/certs/tls.key", "Private key for TLS")
    flag.StringVar(&tlsCert, "tls-crt", "/etc/webhook/certs/tls.crt", "TLS certificate")
    flag.Parse()
    slog.Info("loading certs..")
    certs, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
    if err != nil {
        slog.Error("unable to load certs", "error", err)
        return // Added error handling
    }

    http.HandleFunc("/validate", validate)

    slog.Info("successfully loaded certs. Starting server...", "port", port)
    server := http.Server{
        Addr: fmt.Sprintf(":%d", port),
        TLSConfig: &tls.Config{
            Certificates: []tls.Certificate{certs},
        },
    }

    if err := server.ListenAndServeTLS("", ""); err != nil {
        log.Panic(err)
    }
}