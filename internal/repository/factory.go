package repository

import (
	"fmt"
)

type Repository interface {
	GetContent() ([]map[string]interface{}, error)
	DownloadContents(repoContent []map[string]interface{}, file string) error
}

func NewRepository(Provider string, url string, token string) (Repository, error) {
	switch Provider {
	case "GITHUB":
		return Github{Url: url, Token: token}, nil
	default:
		return nil, fmt.Errorf("provider %s not supported", Provider)
	}
}
