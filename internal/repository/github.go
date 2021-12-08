package repository

import (
	"fmt"
	"gopkg.in/resty.v1"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"path/filepath"
)

type Github struct {
	Url   string
	Token string
}

func (g Github) GetContent() ([]map[string]interface{}, error) {
	client := resty.New()
	resp, err := client.R().SetHeader("Authorization", fmt.Sprintf("Token %s", g.Token)).Get(g.Url)
	if err != nil {
		return nil, err
	}
	var contentList []map[string]interface{}
	err = json.Unmarshal(resp.Body(), &contentList)
	if err != nil {
		return nil, err
	}
	return contentList, nil
}
func (g Github) DownloadContents(repoContent []map[string]interface{}, file string) error {
	fmt.Println("Start download content")
	for _, value := range repoContent {
		if value["download_url"] != nil {
			err := g.downloadContent(fmt.Sprintf("%s/%s", file, value["path"].(string)), value["url"].(string), g.Token)
			if err != nil {
				return err
			}
		} else {
			contents, err := g.getUrlContent(value["url"].(string), g.Token)
			if err != nil {
				return err
			}
			err = g.DownloadContents(contents, file)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (g Github) downloadContent(path string, gitUrl string, token string) error {
	// Create the file
	fmt.Println("Downloading content", path)
	fmt.Println("Download url", gitUrl)
	client := resty.New()
	resp, err := client.R().SetHeader("Authorization", fmt.Sprintf("Token %s", token)).Get(gitUrl)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	file, err := os.Create(path)
	if os.IsNotExist(err) {
		pathDir := filepath.Dir(path)
		fmt.Println(pathDir)
		os.MkdirAll(pathDir, 0766)
		file, err = os.Create(path)
		os.Create(path)
	}
	if err != nil {
		return err
	}
	bytesResponse, err := json.Marshal(resp.Body())
	if err != nil {
		fmt.Println("Error: ", err)
		return err
	}
	_, err = file.Write(bytesResponse)
	if err != nil {
		return err
	}
	return nil
}

func (g Github) getUrlContent(url string, token string) ([]map[string]interface{}, error) {
	client := resty.New()
	resp, err := client.R().SetHeader("Authorization", fmt.Sprintf("token %s", token)).Get(url)
	if err != nil {
		return nil, err
	}
	var contentList []map[string]interface{}
	err = json.Unmarshal(resp.Body(), &contentList)
	if err != nil {
		return nil, err
	}
	return contentList, nil
}
