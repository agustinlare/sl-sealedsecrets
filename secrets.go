package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"gopkg.in/yaml.v2"
)

type InputSecret struct {
	APIVersion  string            `yaml:"apiVersion"`
	Kind        string            `yaml:"kind"`
	Metadata    Metadata          `yaml:"metadata"`
	StringData  map[string]string `yaml:"stringData"`
	Type        string            `yaml:"type"`
	UnknownKeys map[string]interface{}
}

type Metadata struct {
	Name        string `yaml:"name"`
	Namespace   string `yaml:"namespace"`
	ProjectKey  string `yaml:"projectkey"`
	Repository  string `yaml:"repository"`
	UnknownKeys map[string]interface{}
}

type SealedSecrets struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		CreationTimestamp any    `json:"creationTimestamp"`
	} `json:"metadata"`
	Spec struct {
		Template struct {
			Metadata struct {
				Name              string `json:"name"`
				Namespace         string `json:"namespace"`
				CreationTimestamp any    `json:"creationTimestamp"`
			} `json:"metadata"`
			Type string `json:"type"`
		} `json:"template"`
		EncryptedData map[string]string `json:"encryptedData"`
	} `json:"spec"`
}

func lintSecretYaml(secretPath string) error {
	secret, err := parseYAML(secretPath)
	if err != nil {
		return fmt.Errorf("unable to parse '%s' file: %v", secretPath, err)
	}

	// secret.UnknownKeys = getUnknownKeys(secret, reflect.TypeOf(secret))
	// secret.Metadata.UnknownKeys = getUnknownKeys(secret.Metadata, reflect.TypeOf(secret.Metadata))
	// if len(secret.Metadata.UnknownKeys) > 0 {
	// 	return fmt.Errorf("unknown keys found in the metadata section of the YAML file: %v", secret.Metadata.UnknownKeys)
	// }

	if secret.Metadata.Name == "" {
		return fmt.Errorf("metadata.name does not exist")
	}

	if secret.Metadata.Namespace == "" {
		return fmt.Errorf("metadata.namespace does not exist")
	}

	if secret.Metadata.ProjectKey == "" {
		return fmt.Errorf("metadata.projectKey does not exist")
	}

	if secret.Metadata.Repository == "" {
		return fmt.Errorf("metadata.repository does not exist")
	}

	parts := strings.Split(secret.Metadata.Namespace, "-")
	lastSegment := parts[len(parts)-1]
	if lastSegment != "pro" && lastSegment != "dev" && lastSegment != "tst" && lastSegment != "pre" {
		return fmt.Errorf("unable to scrape env from metadata.namespace")
	}

	itsBatch := checkBatch(secret.Metadata.Namespace)
	projectLow := strings.ToLower(secret.Metadata.ProjectKey)
	if itsBatch && projectLow[len(projectLow)-5:] != "batch" {
		return fmt.Errorf("found inconsistency for batch projectKey: %s namespace: %s should match as batch", secret.Metadata.ProjectKey, secret.Metadata.Namespace)
	}

	return nil
}

// func getUnknownKeys(obj interface{}, objType reflect.Type) map[string]interface{} {
// 	knownKeys := make(map[string]bool)
// 	for i := 0; i < objType.NumField(); i++ {
// 		field := objType.Field(i)
// 		if tag, ok := field.Tag.Lookup("yaml"); ok {
// 			knownKeys[tag] = true
// 		}
// 	}

// 	unknownKeys := make(map[string]interface{})
// 	objValue := reflect.ValueOf(obj)
// 	objKind := objValue.Kind()
// 	if objKind == reflect.Ptr {
// 		objValue = objValue.Elem()
// 		objKind = objValue.Kind()
// 	}

// 	if objKind == reflect.Struct {
// 		for i := 0; i < objValue.NumField(); i++ {
// 			field := objValue.Type().Field(i)
// 			if !knownKeys[field.Name] {
// 				unknownKeys[field.Name] = objValue.Field(i).Interface()
// 			}
// 		}
// 	}

// 	return unknownKeys
// }

func checkBatch(s string) bool {
	var resp bool
	if strings.Contains(s, "batch") {
		resp = true
	} else {
		resp = false
	}

	return resp
}

func parseYAML(secretPath string) (InputSecret, error) {
	yamlData, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return InputSecret{}, err
	}

	var secret InputSecret
	err = yaml.Unmarshal(yamlData, &secret)
	if err != nil {
		return InputSecret{}, err
	}

	return secret, nil
}

func sealed(secretPath string, itsProd bool) (SealedSecrets, error) {
	var sealedData SealedSecrets
	err := loginWithToken(itsProd)
	if err != nil {
		return sealedData, err
	}

	var sealedSecretCert []byte
	if itsProd {
		sealedSecretCert, err = base64.StdEncoding.DecodeString(os.Getenv("SS_CERT_PRO"))
	} else {
		sealedSecretCert, err = base64.StdEncoding.DecodeString(os.Getenv("SS_CERT_NOP"))
	}
	if err != nil {
		return sealedData, fmt.Errorf("error decoding certificate: %s", err)
	}

	certPath := "/tmp/sealed-secrets.crt"
	err = ioutil.WriteFile(certPath, sealedSecretCert, 0644)
	if err != nil {
		return sealedData, fmt.Errorf("error writing certificate to file: %s", err)
	}

	cmd := exec.Command("kubeseal",
		"--cert", certPath,
		"--controller-name=sealed-secrets",
		"--controller-namespace", "sealed-secrets",
	)
	secretFileData, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return sealedData, fmt.Errorf("error reading secret file: %s", err)
	}

	cmd.Stdin = bytes.NewReader(secretFileData)
	output, err := cmd.Output()
	if err != nil {
		return sealedData, fmt.Errorf("error running kubeseal command: %s", err)
	}

	sealPath := "/tmp/" + generateFilename("sealed.json")
	err = ioutil.WriteFile(sealPath, output, 0644)
	if err != nil {
		return sealedData, fmt.Errorf("error writing sealed secret to file: %s", err)
	}

	err = json.Unmarshal(output, &sealedData)
	if err != nil {
		return sealedData, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return sealedData, nil
}

func loginWithToken(itsProd bool) error {
	var serverURL, token string
	if itsProd {
		serverURL = os.Getenv("OC_SERVER_NOP")
		token = os.Getenv("OC_TOKEN_PRO")
	} else {
		serverURL = os.Getenv("OC_SERVER_NOP")
		token = os.Getenv("OC_TOKEN_NOP")
	}

	cmd := exec.Command("oc", "login", "--token", token, "--server", serverURL)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'oc' command: %v", err)
	}

	return nil
}
