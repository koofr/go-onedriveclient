package onedriveclient

import (
	"encoding/json"
	"github.com/koofr/go-httpclient"
)

const (
	ErrorCodeItemNotFound      = "itemNotFound"
	ErrorCodeNameAlreadyExists = "nameAlreadyExists"
)

type OneDriveErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type OneDriveError struct {
	Err             OneDriveErrorDetails `json:"error"`
	HttpClientError *httpclient.InvalidStatusError
}

func (e *OneDriveError) Error() string {
	return e.Err.Message
}

func IsOneDriveError(err error) (oneDriveErr *OneDriveError, ok bool) {
	if ode, ok := err.(*OneDriveError); ok {
		return ode, true
	} else {
		return nil, false
	}
}

func HandleError(err error) error {
	if ise, ok := httpclient.IsInvalidStatusError(err); ok {
		oneDriveErr := &OneDriveError{}

		if ise.Headers.Get("Content-Type") == "application/json" {
			if jsonErr := json.Unmarshal([]byte(ise.Content), &oneDriveErr); jsonErr != nil {
				oneDriveErr.Err.Code = "unknown"
				oneDriveErr.Err.Message = ise.Content
			}
		} else {
			oneDriveErr.Err.Code = "unknown"
			oneDriveErr.Err.Message = ise.Content
		}

		oneDriveErr.HttpClientError = ise

		return oneDriveErr
	} else {
		return err
	}
}
