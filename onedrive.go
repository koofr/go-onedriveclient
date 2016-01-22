package onedriveclient

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/koofr/go-httpclient"
	"github.com/koofr/go-ioutils"
)

const (
	DefaultMaxFragmentSize = 60 * 1024 * 1024
)

type OneDrive struct {
	ApiClient       *httpclient.HTTPClient
	Auth            *OneDriveAuth
	MaxFragmentSize int64
}

func NewOneDrive(auth *OneDriveAuth) (c *OneDrive) {
	apiBaseUrl, _ := url.Parse("https://api.onedrive.com/v1.0")
	apiHttpClient := httpclient.New()
	apiHttpClient.BaseURL = apiBaseUrl

	c = &OneDrive{
		ApiClient:       apiHttpClient,
		Auth:            auth,
		MaxFragmentSize: DefaultMaxFragmentSize,
	}

	return c
}

func (c *OneDrive) HandleError(err error) error {
	return HandleError(err)
}

func (c *OneDrive) Request(request *httpclient.RequestData) (res *http.Response, err error) {
	token, err := c.Auth.ValidToken()
	if err != nil {
		return nil, err
	}

	if request.Headers == nil {
		request.Headers = http.Header{}
	}

	request.Headers.Set("Authorization", "Bearer "+token)

	res, err = c.ApiClient.Request(request)

	if err != nil {
		return res, c.HandleError(err)
	}

	return res, nil
}

func (c *OneDrive) Drive() (drive *Drive, err error) {
	req := &httpclient.RequestData{
		Method:         "GET",
		Path:           "/drive",
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &drive,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return drive, nil
}

func (c *OneDrive) ItemsGet(address Address) (item *Item, err error) {
	req := &httpclient.RequestData{
		Method:         "GET",
		Path:           address.String(),
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &item,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (c *OneDrive) ItemsUpdate(address Address, itemUpdate *ItemUpdateBody) (item *Item, err error) {
	req := &httpclient.RequestData{
		Method:         "PATCH",
		Path:           address.String(),
		ExpectedStatus: []int{http.StatusOK},
		ReqEncoding:    httpclient.EncodingJSON,
		ReqValue:       itemUpdate,
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &item,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (c *OneDrive) ItemsDelete(address Address) (err error) {
	req := &httpclient.RequestData{
		Method:         "DELETE",
		Path:           address.String(),
		ExpectedStatus: []int{http.StatusNoContent},
		RespConsume:    true,
	}

	_, err = c.Request(req)

	if err != nil {
		return err
	}

	return nil
}

func (c *OneDrive) ItemsCreate(address Address, body *ItemCreateBody) (item *Item, err error) {
	req := &httpclient.RequestData{
		Method:         "POST",
		Path:           address.Subpath("/children").String(),
		ExpectedStatus: []int{http.StatusCreated},
		ReqEncoding:    httpclient.EncodingJSON,
		ReqValue:       body,
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &item,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (c *OneDrive) ItemsChildren(address Address, link string) (res *ItemCollectionPage, err error) {
	req := &httpclient.RequestData{
		Method:         "GET",
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &res,
	}

	if link != "" {
		req.FullURL = link
	} else {
		req.Path = address.Subpath("/children").String()
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *OneDrive) ItemsCopy(address Address, body *ItemCopyBody) (monitorUrl string, err error) {
	headers := make(http.Header)
	headers.Set("Prefer", "respond-async")

	req := &httpclient.RequestData{
		Method:         "POST",
		Path:           address.Subpath("/action.copy").String(),
		Headers:        headers,
		ExpectedStatus: []int{http.StatusAccepted},
		ReqEncoding:    httpclient.EncodingJSON,
		ReqValue:       body,
		RespConsume:    true,
	}

	res, err := c.Request(req)

	if err != nil {
		return "", err
	}

	monitorUrl = res.Header.Get("Location")

	return monitorUrl, nil
}

func (c *OneDrive) ItemsCopyStatus(monitorUrl string) (status *AsyncOperationStatus, item *Item, err error) {
	req := &httpclient.RequestData{
		Method:          "GET",
		FullURL:         monitorUrl,
		ExpectedStatus:  []int{http.StatusAccepted, http.StatusSeeOther},
		IgnoreRedirects: true,
	}

	res, err := c.Request(req)

	if err != nil {
		return nil, nil, err
	}

	if res.StatusCode == http.StatusAccepted {
		defer res.Body.Close()

		buf, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return nil, nil, err
		}

		err = json.Unmarshal(buf, &status)

		if err != nil {
			return nil, nil, err
		}

		return status, nil, nil
	} else {
		defer res.Body.Close()

		location := res.Header.Get("Location")

		req := &httpclient.RequestData{
			Method:         "GET",
			FullURL:        location,
			ExpectedStatus: []int{http.StatusOK},
			RespEncoding:   httpclient.EncodingJSON,
			RespValue:      &item,
		}

		_, err = c.Request(req)

		if err != nil {
			return nil, nil, err
		}

		return nil, item, nil
	}
}

func (c *OneDrive) ItemsCopyAwait(monitorUrl string) (item *Item, err error) {
	for i := 0; i < 100; i++ {
		time.Sleep(500 * time.Millisecond)

		status, item, err := c.ItemsCopyStatus(monitorUrl)
		if err != nil {
			return nil, err
		}
		if item != nil {
			return item, nil
		}
		if status.Status == AsyncOperationStatusFailed {
			return nil, fmt.Errorf("Copy failed.")
		}
	}

	return nil, fmt.Errorf("Copy progress too long.")
}

func (c *OneDrive) ItemsDelta(address Address, link string, token string) (res *DeltaCollectionPage, err error) {
	req := &httpclient.RequestData{
		Method:         "GET",
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &res,
	}

	if link != "" {
		req.FullURL = link
	} else {
		req.Path = address.Subpath("/view.delta").String()

		if token != "" {
			req.Params = make(url.Values)
			req.Params.Set("token", token)
		}
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *OneDrive) ItemsContent(address Address, span *ioutils.FileSpan) (reader io.ReadCloser, size int64, err error) {
	req := &httpclient.RequestData{
		Method:          "GET",
		Path:            address.Subpath("/content").String(),
		ExpectedStatus:  []int{http.StatusFound},
		IgnoreRedirects: true,
		RespConsume:     true,
	}

	res, err := c.Request(req)

	if err != nil {
		return nil, 0, err
	}

	location := res.Header.Get("Location")

	req = &httpclient.RequestData{
		Method:          "GET",
		FullURL:         location,
		ExpectedStatus:  []int{http.StatusOK, http.StatusPartialContent},
		IgnoreRedirects: true,
	}

	if span != nil {
		req.Headers = make(http.Header)
		req.Headers.Set("Range", fmt.Sprintf("bytes=%d-%d", span.Start, span.End))
	}

	res, err = c.Request(req)

	if err != nil {
		return nil, 0, err
	}

	return res.Body, res.ContentLength, nil
}

func (c *OneDrive) ItemsUploadCreateSession(address Address, body *CreateSessionBody) (uploadSession *UploadSession, err error) {
	uploadSession = &UploadSession{}

	var path string

	if address.Type == AddressTypeId {
		path = address.Subpath(":/" + body.Item.Name + ":/upload.createSession").String()
	} else {
		path = address.Subpath("/upload.createSession").String()
	}

	req := &httpclient.RequestData{
		Method:         "POST",
		Path:           path,
		ExpectedStatus: []int{http.StatusOK, http.StatusPartialContent},
		ReqEncoding:    httpclient.EncodingJSON,
		ReqValue:       body,
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &uploadSession,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return uploadSession, nil
}

func (c *OneDrive) ItemsUploadSessionAppend(uploadSession *UploadSession, content io.Reader, start int64, end int64, size int64) (err error) {
	uploadHeaders := http.Header{}
	uploadHeaders.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))

	req := &httpclient.RequestData{
		Method:         "PUT",
		FullURL:        uploadSession.UploadUrl,
		Headers:        uploadHeaders,
		ExpectedStatus: []int{http.StatusAccepted},
		ReqReader:      content,
	}

	_, err = c.Request(req)

	if err != nil {
		return err
	}

	return nil
}

func (c *OneDrive) ItemsUploadSessionFinish(uploadSession *UploadSession, content io.Reader, start int64, end int64, size int64) (item *Item, err error) {
	item = &Item{}

	uploadHeaders := http.Header{}
	uploadHeaders.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))

	req := &httpclient.RequestData{
		Method:         "PUT",
		FullURL:        uploadSession.UploadUrl,
		Headers:        uploadHeaders,
		ExpectedStatus: []int{http.StatusOK, http.StatusCreated},
		ReqReader:      content,
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &item,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (c *OneDrive) ItemsUpload(address Address, name string, nameConflictBehavior string, content io.Reader, size int64) (item *Item, err error) {
	createSessionBody := &CreateSessionBody{
		Item: ChunkedUploadSessionDescriptor{
			NameConflictBehavior: nameConflictBehavior,
			Name:                 name,
		},
	}

	uploadSession, err := c.ItemsUploadCreateSession(address, createSessionBody)
	if err != nil {
		return nil, err
	}

	reader := ioutils.NewEofReader(content)

	uploaded := int64(0)

	for !reader.Eof {
		start := uploaded
		partSize := c.MaxFragmentSize
		last := false

		if left := size - uploaded; left <= partSize {
			partSize = left
			last = true
		}

		end := start + partSize - 1

		uploaded += partSize

		partReader := io.LimitReader(reader, partSize)

		if last {
			item, err = c.ItemsUploadSessionFinish(uploadSession, partReader, start, end, size)
			if err != nil {
				return nil, err
			}

			return item, nil
		} else {
			err = c.ItemsUploadSessionAppend(uploadSession, partReader, start, end, size)
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("Invalid state")
}
