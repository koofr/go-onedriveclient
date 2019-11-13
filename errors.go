package onedriveclient

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/koofr/go-httpclient"
)

const (
	ErrorCodeItemNotFound      = "itemNotFound"
	ErrorCodeNameAlreadyExists = "nameAlreadyExists"
)

var ErrCompletedNoItem = errors.New("Async task completed but no item")

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

func IsErrorResync(err error) bool {
	if ode, ok := IsOneDriveError(err); ok {
		return ode.Err.Code == "resyncRequired" ||
			ode.Err.Code == "ResyncChangesApplyDifferences" ||
			ode.Err.Code == "ResyncChangesUploadDifferences" ||
			ode.Err.Code == "resyncChangesApplyDifferences" ||
			ode.Err.Code == "resyncChangesUploadDifferences" ||
			ode.Err.Code == "resyncApplyDifferences" ||
			ode.Err.Code == "resyncUploadDifferences" ||
			ode.Err.Code == ErrorCodeItemNotFound ||
			ode.HttpClientError.Got == http.StatusGone
	}

	return false
}

func HandleError(err error) error {
	if ise, ok := httpclient.IsInvalidStatusError(err); ok {
		oneDriveErr := &OneDriveError{}

		if strings.HasPrefix(ise.Headers.Get("Content-Type"), "application/json") {
			if jsonErr := json.Unmarshal([]byte(ise.Content), &oneDriveErr); jsonErr != nil {
				oneDriveErr.Err.Code = "unknown"
				oneDriveErr.Err.Message = ise.Content
			}
		} else {
			oneDriveErr.Err.Code = "unknown"
			oneDriveErr.Err.Message = ise.Content
		}

		if oneDriveErr.Err.Message == "" {
			oneDriveErr.Err.Message = ise.Error()
		}

		oneDriveErr.HttpClientError = ise

		return oneDriveErr
	} else {
		return err
	}
}
