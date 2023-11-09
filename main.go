package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// var Info = log.New(os.Stdout, "\u001b[34mINFO: \u001B[0m", log.LstdFlags|log.Lshortfile)
// var Warning = log.New(os.Stdout, "\u001b[33mWARNING: \u001B[0m", log.LstdFlags|log.Lshortfile)
// var Error = log.New(os.Stdout, "\u001b[31mERROR: \u001b[0m", log.LstdFlags|log.Lshortfile)
// var Debug = log.New(os.Stdout, "\u001b[36mDEBUG: \u001B[0m", log.LstdFlags|log.Lshortfile)

func Router(r *gin.Engine) {
	r.GET("/", SecretUpload)
	r.POST("/", SecretUpload)
}

func main() {
	// initEnvs()

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Static("/css", "./static/css")
	r.Static("/img", "./static/img")
	r.StaticFile("/favicon.ico", "./img/favicon.ico")
	r.LoadHTMLGlob("templates/**/*")

	Router(r)

	log.Println("Server started")
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func SecretUpload(c *gin.Context) {
	if c.Request.Method == "GET" {
		c.HTML(
			http.StatusOK,
			"views/index.html",
			gin.H{},
		)
	}

	if c.Request.Method == "POST" {
		var headers gin.H
		fmt.Println(headers)

		// Get secret file
		file, err := c.FormFile("file")
		if err != nil {
			_ = errResponse(fmt.Errorf("failed to create file: %s", err), c)
			return
		}

		// Save the secret file
		filePath := "/tmp/" + generateFilename(file.Filename)
		err = c.SaveUploadedFile(file, filePath)
		if err != nil {
			_ = errResponse(fmt.Errorf("failed to create file: %s", err), c)
			return
		}

		// Lint secret file for correct keys
		log.Printf("New filecreated: '%s' ", filePath)
		err = lintSecretYaml(filePath)
		if err != nil {
			_ = errResponse(err, c)
			return
		}

		// Get params from Yaml
		secret, _ := parseYAML(filePath)
		parts := strings.Split(secret.Metadata.Namespace, "-")
		environment := parts[len(parts)-1]
		var itsProd bool

		if environment == "pro" {
			itsProd = true
		} else if environment == "dev" || environment == "tst" || environment == "pre" {
			itsProd = false
		}

		var sealedData SealedSecrets
		sealedData, err = sealed(filePath, itsProd)
		if err != nil {
			_ = errResponse(err, c)
			return
		}

		// repoPath, err := cloneRepository(environment, secret.Metadata.ProjectKey, secret.Metadata.Repository)
		// if err != nil {
		// 	_ = errResponse(err, c)
		// 	return
		// }

		repoPath := "/tmp/encuesta-satisfaccion-cliente-2854192321"

		err = updateSecretYaml(repoPath, secret, sealedData)
		if err != nil {
			_ = errResponse(err, c)
			return
		}

		fmt.Println(repoPath, itsProd, sealedData)
	}
}
