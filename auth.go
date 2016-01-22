package onedriveclient

import (
	"encoding/json"
	"github.com/koofr/go-httpclient"
	"net/http"
	"net/url"
	"time"
)

const (
	InvalidGrantError = "invalid_grant"
)

type RefreshResp struct {
	ExpiresIn   int64  `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

type RefreshRespError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type OneDriveAuth struct {
	ClientId     string
	ClientSecret string
	RedirectUri  string
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

func (a *OneDriveAuth) ValidToken() (token string, err error) {
	if time.Now().Unix() > a.ExpiresAt.Unix() {
		data := url.Values{}
		data.Set("grant_type", "refresh_token")
		data.Set("client_id", a.ClientId)
		data.Set("client_secret", a.ClientSecret)
		data.Set("redirect_uri", a.RedirectUri)
		data.Set("refresh_token", a.RefreshToken)

		var respVal RefreshResp

		_, err = httpclient.DefaultClient.Request(&httpclient.RequestData{
			Method:         "POST",
			FullURL:        "https://login.live.com/oauth20_token.srf",
			ExpectedStatus: []int{http.StatusOK},
			ReqEncoding:    httpclient.EncodingForm,
			ReqValue:       data,
			RespEncoding:   httpclient.EncodingJSON,
			RespValue:      &respVal,
		})

		if err != nil {
			err = HandleError(err)

			if ode, ok := IsOneDriveError(err); ok {
				refreshErr := &RefreshRespError{}
				if jsonErr := json.Unmarshal([]byte(ode.Err.Message), &refreshErr); jsonErr == nil {
					ode.Err.Code = refreshErr.Error
					ode.Err.Message = refreshErr.ErrorDescription
				}
			}

			return "", err
		}

		a.AccessToken = respVal.AccessToken
		a.ExpiresAt = time.Now().Add(time.Duration(respVal.ExpiresIn) * time.Second)
	}

	token = a.AccessToken

	return token, nil
}
