package onedriveclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/koofr/go-httpclient"
)

const (
	InvalidGrantError = "invalid_grant"
)

type RefreshResp struct {
	ExpiresIn    int64  `json:"expires_in"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRespError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type OneDriveAuth struct {
	ClientId       string
	ClientSecret   string
	RedirectUri    string
	AccessToken    string
	RefreshToken   string
	ExpiresAt      time.Time
	OnTokenRefresh func(ctx context.Context)
	IsGraph        bool
	HTTPClient     *httpclient.HTTPClient

	mutex sync.Mutex
}

func (a *OneDriveAuth) ValidToken(ctx context.Context) (token string, err error) {
	if time.Now().Unix() > a.ExpiresAt.Add(-5*time.Minute).Unix() {
		err = a.UpdateRefreshToken(ctx)
		if err != nil {
			return "", err
		}
	}

	token = a.AccessToken

	return token, nil
}

func (a *OneDriveAuth) UpdateRefreshToken(ctx context.Context) (err error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", a.ClientId)
	data.Set("client_secret", a.ClientSecret)
	data.Set("redirect_uri", a.RedirectUri)
	data.Set("refresh_token", a.RefreshToken)

	var respVal RefreshResp

	fullURL := "https://login.live.com/oauth20_token.srf"

	if a.IsGraph {
		fullURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
	}

	client := a.HTTPClient
	if client == nil {
		client = httpclient.DefaultClient
	}

	_, err = client.Request(&httpclient.RequestData{
		Context:        ctx,
		Method:         "POST",
		FullURL:        fullURL,
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

		return err
	}

	a.AccessToken = respVal.AccessToken
	a.RefreshToken = respVal.RefreshToken
	a.ExpiresAt = time.Now().Add(time.Duration(respVal.ExpiresIn) * time.Second)

	if a.OnTokenRefresh != nil {
		a.OnTokenRefresh(ctx)
	}

	return nil
}
