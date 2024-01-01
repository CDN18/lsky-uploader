package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Active string `yaml:"active"`
	Users  []UserConfig
}

type UserConfig struct {
	Name       string `yaml:"name"`
	Host       string `yaml:"host"`
	Token      string `yaml:"token"`
	Strategy   int    `yaml:"strategy"`
	Album      int    `yaml:"album"`
	Permission int    `yaml:"permission"`
}

type Response struct {
	Data struct {
		Links struct {
			URL string `json:"url"`
		} `json:"links"`
	} `json:"data"`
}

func main() {
	config := readConfig()

	var activeUserConfig UserConfig
	for _, userConfig := range config.Users {
		if userConfig.Name == config.Active {
			activeUserConfig = userConfig
			break
		}
	}

	for _, filePath := range os.Args[1:] {
		err := uploadFile(filePath, activeUserConfig)
		if err != nil {
			fmt.Println("ERROR:", err)
		}
	}
}

func readConfig() Config {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exeDir := filepath.Dir(exePath)

	configFile, err := os.ReadFile(filepath.Join(exeDir, "config.yml"))
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(configFile, &config)
	if err != nil {
		panic(err)
	}

	return config
}

func uploadFile(filePath string, userConfig UserConfig) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	io.Copy(part, file)

	if userConfig.Strategy != 0 {
		writer.WriteField("strategy_id", fmt.Sprintf("%d", userConfig.Strategy))
	}
	if userConfig.Album != 0 {
		writer.WriteField("album_id", fmt.Sprintf("%d", userConfig.Album))
	}
	if userConfig.Permission != 0 {
		writer.WriteField("permission", fmt.Sprintf("%d", userConfig.Permission))
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", userConfig.Host+"/api/v1/upload", body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", userConfig.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bad status: %s", respBody)
	}

	var response Response
	json.NewDecoder(resp.Body).Decode(&response)

	fmt.Println(response.Data.Links.URL)

	return nil
}
