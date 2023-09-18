package onedriveclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/koofr/go-httpclient"
	"github.com/koofr/go-ioutils"
	"github.com/koofr/go-pathutils"
)

const (
	DefaultMaxFragmentSize = 60 * 1024 * 1024
)

type OneDrive struct {
	ApiClient                *httpclient.HTTPClient
	Auth                     *OneDriveAuth
	MaxFragmentSize          int64
	DriveId                  string
	IsGraph                  bool
	UnusedFilenameMaxRetries int
}

func NewOneDrive(auth *OneDriveAuth) (c *OneDrive) {
	apiBaseUrl, _ := url.Parse("https://api.onedrive.com/v1.0")
	apiHttpClient := httpclient.New()
	apiHttpClient.BaseURL = apiBaseUrl

	c = &OneDrive{
		ApiClient:                apiHttpClient,
		Auth:                     auth,
		MaxFragmentSize:          DefaultMaxFragmentSize,
		DriveId:                  "",
		IsGraph:                  false,
		UnusedFilenameMaxRetries: 100,
	}

	return c
}

func NewOneDriveGraph(auth *OneDriveAuth, driveId string) (c *OneDrive) {
	apiBaseUrl, _ := url.Parse("https://graph.microsoft.com/v1.0/me")
	apiHttpClient := httpclient.New()
	apiHttpClient.BaseURL = apiBaseUrl

	c = &OneDrive{
		ApiClient:                apiHttpClient,
		Auth:                     auth,
		MaxFragmentSize:          DefaultMaxFragmentSize,
		DriveId:                  driveId,
		IsGraph:                  true,
		UnusedFilenameMaxRetries: 100,
	}

	return c
}

func (c *OneDrive) HandleError(err error) error {
	return HandleError(err)
}

func (c *OneDrive) Request(request *httpclient.RequestData) (res *http.Response, err error) {
	authCtx := request.Context
	if authCtx == nil {
		authCtx = context.Background()
	}

	token, err := c.Auth.ValidToken(authCtx)
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

func (c *OneDrive) RequestUnauthorized(request *httpclient.RequestData) (res *http.Response, err error) {
	res, err = c.ApiClient.Request(request)

	if err != nil {
		return res, c.HandleError(err)
	}

	return res, nil
}

func (c *OneDrive) Drive(ctx context.Context) (drive *Drive, err error) {
	path := "/drive"

	if c.IsGraph {
		path = "/drives/" + c.DriveId
	}

	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "GET",
		Path:           path,
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

func (c *OneDrive) ItemsGet(ctx context.Context, address Address) (item *Item, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "GET",
		Path:           address.String(c.DriveId),
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

func (c *OneDrive) ItemsGetHead(ctx context.Context, address Address) (exists bool, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "HEAD",
		Path:           address.String(c.DriveId),
		ExpectedStatus: []int{http.StatusOK, http.StatusNotFound},
	}

	res, err := c.Request(req)

	if err != nil {
		return false, err
	}

	return res.StatusCode == http.StatusOK, err
}

func (c *OneDrive) ItemsUpdate(ctx context.Context, address Address, itemUpdate *ItemUpdateBody) (item *Item, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "PATCH",
		Path:           address.String(c.DriveId),
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

func (c *OneDrive) ItemsDelete(ctx context.Context, address Address) (err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "DELETE",
		Path:           address.String(c.DriveId),
		ExpectedStatus: []int{http.StatusNoContent},
		RespConsume:    true,
	}

	_, err = c.Request(req)

	if err != nil {
		return err
	}

	return nil
}

func (c *OneDrive) ItemsCreate(ctx context.Context, address Address, body *ItemCreateBody) (item *Item, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "POST",
		Path:           address.Subpath("/children").String(c.DriveId),
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

func (c *OneDrive) ItemsChildren(ctx context.Context, address Address, link string) (res *ItemCollectionPage, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "GET",
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &res,
	}

	if link != "" {
		req.FullURL = link
	} else {
		req.Path = address.Subpath("/children").String(c.DriveId)
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *OneDrive) ItemsCopy(ctx context.Context, address Address, body *ItemCopyBody) (monitorUrl string, err error) {
	headers := make(http.Header)
	headers.Set("Prefer", "respond-async")

	path := address.Subpath("/action.copy").String(c.DriveId)

	if c.IsGraph {
		path = address.Subpath("/copy").String(c.DriveId)
	}

	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "POST",
		Path:           path,
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

func (c *OneDrive) ItemsCopyStatus(ctx context.Context, monitorUrl string) (status *AsyncOperationStatus, item *Item, err error) {
	if c.IsGraph {
		req := &httpclient.RequestData{
			Context:         ctx,
			Method:          "GET",
			FullURL:         monitorUrl,
			ExpectedStatus:  []int{http.StatusAccepted, http.StatusOK},
			RespValue:       &status,
			RespEncoding:    httpclient.EncodingJSON,
			IgnoreRedirects: true,
		}

		_, err = c.RequestUnauthorized(req)

		if err != nil {
			return nil, nil, err
		}

		return status, nil, nil
	}

	req := &httpclient.RequestData{
		Context:         ctx,
		Method:          "GET",
		FullURL:         monitorUrl,
		ExpectedStatus:  []int{http.StatusAccepted, http.StatusSeeOther},
		IgnoreRedirects: true,
	}

	res, err := c.Request(req)

	if err != nil {
		return nil, nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusAccepted {
		buf, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return nil, nil, err
		}

		err = json.Unmarshal(buf, &status)

		if err != nil {
			return nil, nil, err
		}

		return status, nil, nil
	}

	location := res.Header.Get("Location")

	req = &httpclient.RequestData{
		Context:        ctx,
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

func (c *OneDrive) ItemsCopyAwait(ctx context.Context, monitorUrl string) (item *Item, err error) {
	for i := 0; i < 100; i++ {
		// TODO handle ctx cancelation
		time.Sleep(500 * time.Millisecond)

		status, item, err := c.ItemsCopyStatus(ctx, monitorUrl)
		if err != nil {
			return nil, err
		}
		if item != nil {
			return item, nil
		}
		if status.Status == AsyncOperationStatusFailed {
			return nil, fmt.Errorf("copy failed")
		} else if c.IsGraph && status.Status == AsyncOperationStatusCompleted {
			return nil, ErrCompletedNoItem
		}
	}

	return nil, fmt.Errorf("copy progress too long")
}

func (c *OneDrive) ItemsDelta(ctx context.Context, address Address, link string, token string) (res *DeltaCollectionPage, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "GET",
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &res,
	}

	if link != "" {
		req.FullURL = link
	} else {
		if c.IsGraph {
			req.Path = address.Subpath("/delta").String(c.DriveId)
		} else {
			req.Path = address.Subpath("/view.delta").String(c.DriveId)
		}

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

func (c *OneDrive) ItemsContent(ctx context.Context, address Address, span *ioutils.FileSpan) (reader io.ReadCloser, size int64, err error) {
	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "GET",
		Path:           address.Subpath("/content").String(c.DriveId),
		ExpectedStatus: []int{http.StatusFound, http.StatusOK, http.StatusPartialContent},
	}

	if span != nil {
		req.Headers = make(http.Header)
		req.Headers.Set("Range", fmt.Sprintf("bytes=%d-%d", span.Start, span.End))
	}

	res, err := c.Request(req)

	if err != nil {
		return nil, 0, err
	}

	return res.Body, res.ContentLength, nil
}

func (c *OneDrive) ItemsUploadCreateSession(ctx context.Context, address Address, body BaseCreateSessionBody) (uploadSession *UploadSession, err error) {
	uploadSession = &UploadSession{}

	var path string

	if address.Type == AddressTypeId {
		if c.IsGraph {
			path = address.Subpath(":/" + body.GetName() + ":/createUploadSession").String(c.DriveId)
		} else {
			path = address.Subpath(":/" + body.GetName() + ":/upload.createSession").String(c.DriveId)
		}
	} else {
		if c.IsGraph {
			path = address.Subpath("/createUploadSession").String(c.DriveId)
		} else {
			path = address.Subpath("/upload.createSession").String(c.DriveId)
		}
	}

	if c.IsGraph {
		body.(*GraphCreateSessionBody).Item.Name = ""
	}

	req := &httpclient.RequestData{
		Context:        ctx,
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

func (c *OneDrive) ItemsUploadSessionAppend(ctx context.Context, uploadSession *UploadSession, content io.Reader, start int64, end int64, size int64) (err error) {
	contentLength := (end - start) + 1

	uploadHeaders := http.Header{}
	uploadHeaders.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	uploadHeaders.Set("Content-Length", fmt.Sprintf("%d", (end-start)+1))

	req := &httpclient.RequestData{
		Context:          ctx,
		Method:           "PUT",
		FullURL:          uploadSession.UploadUrl,
		Headers:          uploadHeaders,
		ExpectedStatus:   []int{http.StatusAccepted},
		ReqReader:        content,
		ReqContentLength: contentLength,
	}

	_, err = c.Request(req)

	if err != nil {
		return err
	}

	return nil
}

func (c *OneDrive) ItemsUploadSessionFinish(ctx context.Context, uploadSession *UploadSession, content io.Reader, start int64, end int64, size int64) (item *Item, err error) {
	item = &Item{}

	contentLength := (end - start) + 1

	uploadHeaders := http.Header{}
	uploadHeaders.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	uploadHeaders.Set("Content-Length", fmt.Sprintf("%d", contentLength))

	req := &httpclient.RequestData{
		Context:          ctx,
		Method:           "PUT",
		FullURL:          uploadSession.UploadUrl,
		Headers:          uploadHeaders,
		ExpectedStatus:   []int{http.StatusOK, http.StatusCreated},
		ReqReader:        content,
		ReqContentLength: contentLength,
		RespEncoding:     httpclient.EncodingJSON,
		RespValue:        &item,
	}

	_, err = c.Request(req)

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (c *OneDrive) ItemsUpload(ctx context.Context, address Address, name string, nameConflictBehavior string, content io.Reader, size int64) (item *Item, err error) {

	if size == 0 {
		return c.ItemsUploadSimple(ctx, address, name, nameConflictBehavior, content, size)
	}
	return c.ItemsUploadSession(ctx, address, name, nameConflictBehavior, content, size)
}

func (c *OneDrive) ItemsUploadSimple(ctx context.Context, address Address, name string, nameConflictBehavior string, content io.Reader, size int64) (item *Item, err error) {
	item = &Item{}

	childrenMap := map[string]*Item{}

	fileExists := func(name string) bool {
		_, ok := childrenMap[name]
		return ok
	}

	if nameConflictBehavior != NameConflictBehaviorReplace {
		link := ""

		for {
			page, err := c.ItemsChildren(ctx, address, link)
			if err != nil {
				return nil, err
			}

			if len(page.Value) == 0 {
				break
			}

			for _, item := range page.Value {
				childrenMap[item.Name] = item
				//if item.Name == name {
				//	return nil, fmt.Errorf("Autorename not supported")
				//}
			}

			if page.NextLink == "" {
				break
			}

			link = page.NextLink
		}
	}

	if nameConflictBehavior == NameConflictBehaviorFail && fileExists(name) {
		return nil, fmt.Errorf("File already exists")
	}

	if nameConflictBehavior == NameConflictBehaviorRename {
		name, err = pathutils.UnusedFilename(fileExists, name, c.UnusedFilenameMaxRetries)
		if err != nil {
			return nil, fmt.Errorf("Max autorename attempts reached")
		}
	}

	var path string

	if address.Type == AddressTypeId {
		if c.IsGraph {
			path = address.Subpath(":/" + name + ":/content").String(c.DriveId)
		} else {
			path = address.Subpath(":/" + name + ":/content").String(c.DriveId)
		}
	} else {
		return nil, fmt.Errorf("Not implemented because I have no idea how to properly do it")
	}

	req := &httpclient.RequestData{
		Context:        ctx,
		Method:         "PUT",
		Path:           path,
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

func (c *OneDrive) ItemsUploadSession(ctx context.Context, address Address, name string, nameConflictBehavior string, content io.Reader, size int64) (item *Item, err error) {
	var createSessionBody BaseCreateSessionBody = &CreateSessionBody{
		Item: ChunkedUploadSessionDescriptor{
			NameConflictBehavior: nameConflictBehavior,
			Name:                 name,
		},
	}

	if c.IsGraph {
		createSessionBody = &GraphCreateSessionBody{
			Item: GraphChunkedUploadSessionDescriptor{
				NameConflictBehavior: nameConflictBehavior,
				Name:                 name,
			},
		}
	}

	uploadSession, err := c.ItemsUploadCreateSession(ctx, address, createSessionBody)
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
			item, err = c.ItemsUploadSessionFinish(ctx, uploadSession, partReader, start, end, size)
			if err != nil {
				return nil, err
			}

			return item, nil
		}

		err = c.ItemsUploadSessionAppend(ctx, uploadSession, partReader, start, end, size)
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("invalid state")
}
