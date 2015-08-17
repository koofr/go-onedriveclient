package onedriveclient

import (
	"io"
	"time"
)

type CreateUploadSession struct {
	Item UploadSessionItem `json:"item"`
}

type UploadSessionItem struct {
	NameConflictBehavior string `json:"@name.conflictBehavior"`
	Name                 string `json:"name"`
}

type UploadSession struct {
	UploadUrl string `json:"uploadUrl"`
}

type RefreshResp struct {
	ExpiresIn   int64  `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

type Item struct {
	Id           string    `json:"id"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	Type         string    `json:"type"`
	LastModified time.Time `json:"lastModifiedDateTime"`
	Reader       io.ReadCloser
}
