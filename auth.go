package onedriveclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type OneDriveAuth struct {
	ClientId     string
	ClientSecret string
	RedirectUri  string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (d *OneDriveAuth) ValidToken() (token string, err error) {
	if time.Now().Unix() > d.ExpiresAt.Unix() {
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("client_id", d.ClientId)
		data.Set("client_secret", d.ClientSecret)
		data.Set("redirect_uri", d.RedirectUri)
		data.Set("refresh_token", d.RefreshToken)

		var resp *http.Response

		resp, err = http.PostForm("https://login.live.com/oauth20_token.srf", data)
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
