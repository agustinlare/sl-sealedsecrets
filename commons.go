package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type Links struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

type Config struct {
	Token        string `json:"token"`
	Secretstring string `json:"secretstring"`
	Postgres     struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Dbname   string `json:"dbname"`
		Table    string `json:"table"`
	} `json:"postgres"`
	Lambda struct {
		Dev struct {
			Ec2Stop  string `json:"ec2_stop"`
			Ec2Start string `json:"ec2_start"`
			RdsStop  string `json:"rds_stop"`
			RdsStart string `json:"rds_start"`
			Asc      string `json:"asc"`
		} `json:"dev"`
		Qa struct {
			Ec2Stop  string `json:"ec2_stop"`
			Ec2Start string `json:"ec2_start"`
			RdsStop  string `json:"rds_stop"`
			RdsStart string `json:"rds_start"`
			Asc      string `json:"asc"`
		} `json:"qa"`
	} `json:"lambda"`
	Terraform []struct {
		Name   string `json:"name"`
		Varset string `json:"varset"`
	} `json:"terraform"`
	Links []struct {
		Name string `json:"name"`
		Href string `json:"href"`
	} `json:"links"`
}

func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func generateFilename(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	filename := originalFilename[0 : len(originalFilename)-len(ext)]
	currentTime := getTimeStamp()
	return fmt.Sprintf("%s-%s%s", filename, currentTime, ext)
}

func getTimeStamp() string {
	return time.Now().Format("20060102150405")
}

func errResponse(err error, c *gin.Context) *gin.Context {
	log.Println(err.Error())
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	c.Abort()
	c.Next()
	return c
}
