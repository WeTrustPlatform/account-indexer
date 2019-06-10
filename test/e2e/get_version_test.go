// +build e2e

package e2e

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/WeTrustPlatform/account-indexer/common/config"
	indexerHttp "github.com/WeTrustPlatform/account-indexer/http"
	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	log.Println("Start TestGetVersion")
	url := "http://127.0.0.1:3000/admin/version"
	userName := os.Getenv(indexerHttp.AdminUserName)
	password := os.Getenv(indexerHttp.AdminPassword)
	encoded := base64.StdEncoding.EncodeToString([]byte(userName + ":" + password))
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequest("GET", url, bytes.NewBufferString("{}"))
	assert.Nil(t, err)
	req.Header.Set("Authorization", "Basic "+encoded)
	res, err := httpClient.Do(req)
	assert.Nil(t, err)
	defer res.Body.Close()
	var httpResult config.VersionInfo
	err = json.NewDecoder(res.Body).Decode(&httpResult)
	assert.Nil(t, err)
	log.Printf("TestGetVersion version=%v, buildID=%v, githash=%v \n", httpResult.Version, httpResult.BuildID, httpResult.GitHash)
	log.Println("End TestGetVersion")
}
