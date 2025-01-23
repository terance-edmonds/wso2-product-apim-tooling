package api

import (
	"archive/zip"

	"github.com/wso2/product-apim-tooling/apim-agent/pkg/transformer"
)

// FetchAPIsConf defines the return type of FetchAPIsonEvent
type FetchAPIsConf struct {
	APIs           *[]string
	APIDeployments *[]transformer.Deployment
	APIFiles       map[string]*zip.File
}
