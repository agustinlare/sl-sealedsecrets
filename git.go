package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func cloneRepository(environment string, projectKey string, repository string) (string, error) {
	tempDir, err := ioutil.TempDir("/tmp/", repository+"-")
	if err != nil {
		return "", err
	}

	// defer func() {
	// 	err := os.RemoveAll(tempDir)
	// 	if err != nil {
	// 		log.Println("error removing temporary directory:", err)
	// 		return
	// 	}
	// }()

	// Workaround since the git module wont skip tls verification
	cmd := exec.Command("bash", "git/clone.sh", os.Getenv("GIT_USERNAME"), os.Getenv("GIT_PASSWORD"), projectKey, fmt.Sprintf("%s-config-%s", repository, environment), tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run 'clone' repository: %v", err)
	}

	err = checkoutNewBranch(tempDir, repository)
	if err != nil {
		return "", err
	}

	return tempDir, nil
}

func checkoutNewBranch(tempDir string, repository string) error {
	date := getTimeStamp()

	cmd := exec.Command("git", "checkout", "-b", fmt.Sprintf("feature/update-secret-%s-%s", repository, date))
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch: %v", err)
	}

	return nil
}
