package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"

	"gopkg.in/yaml.v2"
)

// type RepoSecretBatch struct {
// 	LibHelmScdf struct {
// 		SealedSecretsEnabled bool `yaml:"sealedsecretsEnabled"`
// 		SealedSecrets        []struct {
// 			Name    string `yaml:"name"`
// 			Secrets []struct {
// 				Name string `yaml:"name"`
// 				Data string `yaml:"data"`
// 			} `yaml:"secrets"`
// 		} `yaml:"sealedsecrets"`
// 	} `yaml:"lib-helm-scdf"`
// }

type RepoSecretBatch struct {
	LibHelmSCDF struct {
		SealedSecretsEnabled bool           `yaml:"sealedsecretsEnabled"`
		SealedSecrets        []SealedSecret `yaml:"sealedsecrets"`
	} `yaml:"lib-helm-scdf"`
}

type SealedSecret struct {
	Name    string   `yaml:"name"`
	Secrets []Secret `yaml:"secrets"`
}

type Secret struct {
	Name string `yaml:"name"`
	Data string `yaml:"data"`
}

type RepoSecretApp struct {
	LibHelmAMQ struct {
		SealedSecrets map[string]interface{} `yaml:"sealedsecrets"`
	} `yaml:"lib-helm-amq"`
}

func updateSecretYaml(repoPath string, secret InputSecret, sealedData SealedSecrets) error {
	fmt.Println(repoPath, secret, sealedData)
	name := sealedData.Metadata.Name
	encryptedData := sealedData.Spec.EncryptedData
	itsBatch := checkBatch(secret.Metadata.Namespace)
	yamlSecretPath := fmt.Sprintf("%s/cd/secrets.yaml", repoPath)

	data, err := ioutil.ReadFile(yamlSecretPath)
	if err != nil {
		return fmt.Errorf("error reading secrets.yaml: %v", err)
	}

	if itsBatch {
		var fileSecret RepoSecretBatch
		err = yaml.Unmarshal(data, &fileSecret)
		if err != nil {
			return fmt.Errorf("error unmarshaling secrets.yaml: %v", err)
		}

		fileSecret.LibHelmSCDF.SealedSecretsEnabled = true
		var positionKey string
		for i, sealedSecret := range fileSecret.LibHelmSCDF.SealedSecrets {
			if sealedSecret.Name == name {
				positionKey = fmt.Sprintf("%d", i)
				break
			}
		}

		if positionKey == "" {
			positionKey = fmt.Sprintf("%d", len(fileSecret.LibHelmSCDF.SealedSecrets))
			fileSecret.LibHelmSCDF.SealedSecrets = append(fileSecret.LibHelmSCDF.SealedSecrets, SealedSecret{Name: name})
		}

		for i, v := range encryptedData {
			var secretKey string
			newSecretName := i
			newSecretValue := v
			index, _ := strconv.Atoi(positionKey)

			for e, existingSecret := range fileSecret.LibHelmSCDF.SealedSecrets[index].Secrets {
				if existingSecret.Name == newSecretName {
					secretKey = fmt.Sprintf("%d", e)
					break
				}
			}

			if secretKey == "" {
				// secretKey = fmt.Sprintf("%d", len(fileSecret.LibHelmSCDF.SealedSecrets[index].Secrets))
				fileSecret.LibHelmSCDF.SealedSecrets[index].Secrets = append(fileSecret.LibHelmSCDF.SealedSecrets[index].Secrets, Secret{Name: newSecretName, Data: newSecretValue})
			} else {
				ondex, _ := strconv.Atoi(secretKey)
				fileSecret.LibHelmSCDF.SealedSecrets[index].Secrets[ondex].Data = newSecretValue
			}
		}

		yamlData, err := yaml.Marshal(&fileSecret)
		if err != nil {
			return fmt.Errorf("error marshaling secrets to YAML: %v", err)
		}

		err = os.WriteFile(yamlSecretPath, yamlData, 0644)
		if err != nil {
			return fmt.Errorf("error writing secrets.yaml: %v", err)
		}
	} else {
		var fileSecret interface{}
		err := yaml.Unmarshal(data, &fileSecret)
		if err != nil {
			return fmt.Errorf("error unmarshaling secrets.yaml: %v", err)
		}

		if secretMap, ok := fileSecret.(map[interface{}]interface{}); ok {
			// No existe?
			if _, ok := secretMap["enabled"]; !ok {
				secretMap["enabled"] = true
			} else {
				if enabled, ok := secretMap["enabled"].(bool); ok && !enabled {
					secretMap["enabled"] = true
				}
			}

			// Data exist
			for i, v := range encryptedData {
				if _, ok = secretMap["data"]; !ok {
					secretMap["data"] = make(map[interface{}]interface{})
					dataMap := secretMap["data"].(map[interface{}]interface{})
					dataMap[i] = v
				} else {
					dataMap := secretMap["data"].(map[interface{}]interface{})
					dataMap[i] = v
				}
			}
		} else {
			return fmt.Errorf("file it is not formated well")
		}
	}

	cmd := exec.Command("yq", "-i", "--indent", "2", yamlSecretPath)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: %s", err)
	}

	return nil
}
