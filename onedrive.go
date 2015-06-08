package onedriveclient

import (
	"fmt"
	"github.com/koofr/go-httpclient"
	"github.com/koofr/go-ioutils"
	"io"
	"net/http"
	"net/url"
	"path"
)

const (
	DefaultMaxFragmentSize = 60 * 1024 * 1024
)

type OneDrive struct {
	ApiClient       *httpclient.HTTPClient
	Auth            *OneDriveAuth
	MaxFragmentSize int64
}

func NewOneDrive(auth *OneDriveAuth) (d *OneDrive) {
	apiBaseUrl, _ := url.Parse("https://api.onedrive.com/v1.0")
	apiHttpClient := httpclient.New()
	apiHttpClient.BaseURL = apiBaseUrl

	d = &OneDrive{
		ApiClient:       apiHttpClient,
		Auth:            auth,
		MaxFragmentSize: DefaultMaxFragmentSize,
	}

	return
}

func (d *OneDrive) Request(request *httpclient.RequestData) (response *http.Response, err error) {
	token, err := d.Auth.ValidToken()
	if err != nil {
		return
	}

	if request.Headers == nil {
		request.Headers = http.Header{}
	}

	request.Headers.Set("Authorization", "Bearer "+token)

	return d.ApiClient.Request(request)
}

func (d *OneDrive) NormalizePath(pth string) string {
	return path.Clean("/" + pth)
}

func (d *OneDrive) Info(pth string) (item *Item, err error) {
	pth = d.NormalizePath(pth)

	req := &httpclient.RequestData{
		Method:         "GET",
		Path:           "/drive/root:" + pth,
		ExpectedStatus: []int{http.StatusOK},
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &item,
	}

	_, err = d.Request(req)

	if err != nil {
		return
	}

	return
}

func (d *OneDrive) Download(pth string, span *ioutils.FileSpan) (item *Item, err error) {
	pth = d.NormalizePath(pth)

	item, err = d.Info(pth)

	if err != nil {
		return
	}

	req := &httpclient.RequestData{
		Method:          "GET",
		Path:            "/drive/root:" + pth + ":/content",
		ExpectedStatus:  []int{http.StatusFound},
		IgnoreRedirects: true,
		RespConsume:     true,
	}

	res, err := d.Request(req)

	if err != nil {
		return
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

	res, err = d.Request(req)

	if err != nil {
		return
	}

	item.Size = res.ContentLength
	item.Reader = res.Body

	return
}

func (d *OneDrive) Upload(pth string, overwrite bool, content io.Reader, size int64) (item *Item, err error) {
	pth = d.NormalizePath(pth)

	createUploadSession := &CreateUploadSession{
		NameConflictBehavior: "rename",
	}

	if overwrite {
		createUploadSession.NameConflictBehavior = "replace"
	}

	uploadSession := &UploadSession{}

	req := &httpclient.RequestData{
		Method:         "POST",
		Path:           "/drive/root:" + pth + ":/upload.createSession",
		ExpectedStatus: []int{http.StatusOK, http.StatusPartialContent},
		ReqEncoding:    httpclient.EncodingJSON,
		ReqValue:       createUploadSession,
		RespEncoding:   httpclient.EncodingJSON,
		RespValue:      &uploadSession,
	}

	_, err = d.Request(req)

	if err != nil {
		return
	}

	reader := ioutils.NewEofReader(content)

	uploaded := int64(0)

	for !reader.Eof {
		start := uploaded
		partSize := d.MaxFragmentSize
		last := false

		if left := size - uploaded; left <= partSize {
			partSize = left
			last = true
		}

		end := start + partSize - 1

		uploaded += partSize

		partReader := io.LimitReader(reader, partSize)

		uploadHeaders := http.Header{}
		uploadHeaders.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))

		if last {
			req = &httpclient.RequestData{
				Method:         "PUT",
				FullURL:        uploadSession.UploadUrl,
				Headers:        uploadHeaders,
				ExpectedStatus: []int{http.StatusOK, http.StatusCreated},
				ReqReader:      partReader,
				RespEncoding:   httpclient.EncodingJSON,
				RespValue:      &item,
			}

			_, err = d.Request(req)

			if err != nil {
				return
			}

			return
		} else {
			req = &httpclient.RequestData{
				Method:         "PUT",
				FullURL:        uploadSession.UploadUrl,
				Headers:        uploadHeaders,
				ExpectedStatus: []int{http.StatusAccepted},
				ReqReader:      partReader,
			}

			_, err = d.Request(req)

			if err != nil {
				return
			}
		}
	}

	return
}
