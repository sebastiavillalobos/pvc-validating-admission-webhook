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
    "k8s.io/apimachinery/pkg/api/resource"
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
        slog.Error("error reading request body", "error", err)
        return nil, err
    }

    admissionReviewRequest := &admissionv1.AdmissionReview{}
    _, _, err = deserializer.Decode(reqData, nil, admissionReviewRequest)
    if err != nil {
        slog.Error("unable to deserialize request", "error", err)
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

    // Check if admission request is valid
    if admissionReviewRequest.Request == nil {
        err := errors.New("admission review request is nil")
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

    // Check if storage request exists
    storageRequest, exists := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
    if !exists {
        err := errors.New("PVC does not have storage request specified")
        httpError(w, err)
        return
    }

    // Convert storage request to bytes for comparison
    requestedBytes := storageRequest.Value()
    if requestedBytes > maxPVCSize {
        // Create proper admission response with rejection
        admissionResponse := &admissionv1.AdmissionResponse{
            UID:     admissionReviewRequest.Request.UID,
            Allowed: false,
            Result: &metav1.Status{
                Code:    http.StatusForbidden,
                Message: fmt.Sprintf("PVC size %s exceeds the maximum limit of %s", storageRequest.String(), resource.NewQuantity(maxPVCSize, resource.BinarySI).String()),
            },
        }

        admissionReviewResponse := admissionv1.AdmissionReview{
            TypeMeta: metav1.TypeMeta{
                APIVersion: "admission.k8s.io/v1",
                Kind:       "AdmissionReview",
            },
            Response: admissionResponse,
        }

        responseBytes, err := json.Marshal(admissionReviewResponse)
        if err != nil {
            httpError(w, errors.New("unable to marshal rejection response into bytes"))
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK) // Always return 200 OK for admission responses
        w.Write(responseBytes)
        slog.Info("PVC rejected", "name", pvc.ObjectMeta.Name, "requested_size", storageRequest.String(), "max_size", resource.NewQuantity(maxPVCSize, resource.BinarySI).String())
        return
    }

    // Allow the PVC
    admissionResponse := &admissionv1.AdmissionResponse{
        UID:     admissionReviewRequest.Request.UID,
        Allowed: true,
    }

    admissionReviewResponse := admissionv1.AdmissionReview{
        TypeMeta: metav1.TypeMeta{
            APIVersion: "admission.k8s.io/v1",
            Kind:       "AdmissionReview",
        },
        Response: admissionResponse,
    }

    responseBytes, err := json.Marshal(admissionReviewResponse)
    if err != nil {
        httpError(w, errors.New("unable to marshal response into bytes"))
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(responseBytes)
    slog.Info("PVC validated successfully", "name", pvc.ObjectMeta.Name, "requested_size", storageRequest.String())
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