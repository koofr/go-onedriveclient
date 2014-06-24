package onedriveclient

import (
	"encoding/json"
	"fmt"
	"github.com/koofr/go-httpclient"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type OneDrive struct {
	ApiClient *httpclient.HTTPClient
	Auth      *OneDriveAuth
}

type OneDriveAuth struct {
	ClientId     string
	ClientSecret string
	RedirectUri  string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (d *OneDriveAuth) validToken() (token string, err error) {
	if time.Now().Unix() > d.ExpiresAt.Unix() {
		var resp *http.Response
		resp, err = http.PostForm("https://login.live.com/oauth20_token.srf",
			url.Values{
				"grant_type":    {"refresh_token"},
				"client_id":     {d.ClientId},
				"client_secret": {d.ClientSecret},
				"redirect_uri":  {d.RedirectUri},
				"refresh_token": {d.RefreshToken},
			})
		if err != nil {
			return
		}

		if resp.StatusCode != 200 {
			err = fmt.Errorf("Token refresh failed %d: %s", resp.StatusCode, resp.Status)
			return
		}

		var buf []byte
		if buf, err = ioutil.ReadAll(resp.Body); err != nil {
			return
		}

		var respVal RefreshResp
		if err = json.Unmarshal(buf, &respVal); err != nil {
			return
		}

		d.AccessToken = respVal.AccessToken
		d.ExpiresAt = time.Now().Add(time.Duration(respVal.ExpiresIn) * time.Second)
	}
	token = d.AccessToken
	return
}

func NewOneDriveClient(auth OneDriveAuth) *OneDrive {
	apiBaseUrl, _ := url.Parse("https://apis.live.net/v5.0")
	apiHttpClient := httpclient.New()
	apiHttpClient.BaseURL = apiBaseUrl
	return &OneDrive{apiHttpClient, &auth}
}

func (d *OneDrive) authenticationHeader() (hs http.Header, err error) {
	token, err := d.Auth.validToken()
	if err != nil {
		return
	}

	hs = make(http.Header)
	hs.Set("Authorization", "Bearer "+token)
	return
}

func (d *OneDrive) NodeInfo(id string) (info NodeInfo, err error) {
	header, err := d.authenticationHeader()
	if err != nil {
		return
	}

	req := &httpclient.RequestData{
		Method:         "GET",
		Path:           "/" + id,
		Headers:        header,
		ExpectedStatus: []int{200},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &info,
	}
	_, err = d.ApiClient.Request(req)
	if err != nil {
		return
	}
	return
}

func (d *OneDrive) RootInfo() (info NodeInfo, err error) {
	info, err = d.NodeInfo("me/skydrive")
	return
}

func (d *OneDrive) NodeFiles(id string) (files []NodeInfo, err error) {
	header, err := d.authenticationHeader()
	if err != nil {
		return
	}

	var resp NodeFiles
	req := &httpclient.RequestData{
		Method:         "GET",
		Path:           "/" + id + "/files",
		Headers:        header,
		ExpectedStatus: []int{200},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &resp,
	}
	_, err = d.ApiClient.Request(req)
	if err != nil {
		return
	}

	files = resp.Data
	return
}

func (d *OneDrive) ResolvePath(pth string) (id string, err error) {
	root, err := d.RootInfo()
	if err != nil {
		return
	}
	id = root.Id

loopParts:
	for _, part := range pathParts(pth) {
		var files []NodeInfo
		files, err = d.NodeFiles(id)
		if err != nil {
			return
		}
		name := strings.ToLower(part)
		for _, file := range files {
			if strings.ToLower(file.Name) == name {
				id = file.Id
				continue loopParts
			}
		}
		return "", fmt.Errorf("Not found %s in %s", part, files)
	}
	return
}

func pathParts(pth string) []string {
	pth = path.Clean("/" + pth)
	parts := make([]string, 0)
	for pth != "/" {
		var name string
		pth, name = path.Split(pth)
		pth = path.Clean(pth)
		parts = append(parts, name)
	}

	//in-place reverse
	l := len(parts) - 1
	h := len(parts) / 2
	for i := 0; i < h; i++ {
		t := parts[i]
		ii := l - i
		parts[i] = parts[ii]
		parts[ii] = t
	}
	return parts
}
