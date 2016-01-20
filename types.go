package onedriveclient

import (
	"github.com/koofr/go-httpclient"
	"time"
)

const (
	NameConflictBehaviorRename  = "rename"
	NameConflictBehaviorReplace = "replace"
	NameConflictBehaviorFail    = "fail"
)

const (
	AsyncOperationStatusNotStarted    = "notStarted"
	AsyncOperationStatusInProgress    = "inProgress"
	AsyncOperationStatusCompleted     = "completed"
	AsyncOperationStatusUpdating      = "updating"
	AsyncOperationStatusFailed        = "failed"
	AsyncOperationStatusDeletePending = "deletePending"
	AsyncOperationStatusDeleteFailed  = "deleteFailed"
	AsyncOperationStatusWaiting       = "waiting"
)

type ChunkedUploadSessionDescriptor struct {
	NameConflictBehavior string `json:"@name.conflictBehavior"`
	Name                 string `json:"name"`
}

type UploadSession struct {
	UploadUrl          string    `json:"uploadUrl"`
	ExpirationDateTime time.Time `json:"expirationDateTime"`
	NextExpectedRanges []string  `json:"nextExpectedRanges"`
}

type Identity struct {
	DisplayName string `json:"displayName"`
	Id          string `json:"id"`
}

type IdentitySet struct {
	Application Identity `json:"application"`
	User        Identity `json:"user"`
}

type FileSystemInfo struct {
	CreatedDateTime      time.Time `json:"createdDateTime"`
	LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
}

type Folder struct {
	ChildCount int `json:"childCount"`
}

type Hashes struct {
	Crc32Hash string `json:"crc32Hash"`
	Sha1Hash  string `json:"sha1Hash"`
}

type File struct {
	Hashes   Hashes `json:"hashes"`
	MimeType string `json:"mimeType"`
}

type ItemReference struct {
	DriveId string `json:"driveId,omitempty"`
	Id      string `json:"id,omitempty"`
	Path    string `json:"path,omitempty"`
}

type Deleted struct {
	State string `json:"state"`
}

type Quota struct {
	Deleted   int64  `json:"deleted"`
	Remaining int64  `json:"remaining"`
	State     string `json:"state"`
	Total     int64  `json:"total"`
	Used      int64  `json:"used"`
}

type Drive struct {
	Id        string      `json:"id"`
	DriveType string      `json:"driveType"`
	Owner     IdentitySet `json:"owner"`
	Quota     Quota       `json:"quota"`
	// Items
	// Shared
	// Special
}

type Item struct {
	CreatedBy            *IdentitySet    `json:"createdBy,omitempty"`
	CreatedDateTime      time.Time       `json:"createdDateTime,omitempty"`
	CTag                 string          `json:"cTag,omitempty"`
	Description          string          `json:"description,omitempty"`
	ETag                 string          `json:"eTag,omitempty"`
	Id                   string          `json:"id,omitempty"`
	LastModifiedBy       *IdentitySet    `json:"lastModifiedBy,omitempty"`
	LastModifiedDateTime time.Time       `json:"lastModifiedDateTime,omitempty"`
	Name                 string          `json:"name,omitempty"`
	ParentReference      *ItemReference  `json:"parentReference,omitempty"`
	Size                 int64           `json:"size,omitempty"`
	WebURL               string          `json:"webUrl,omitempty"`
	Deleted              *Deleted        `json:"deleted,omitempty"`
	File                 *File           `json:"file,omitempty"`
	FileSystemInfo       *FileSystemInfo `json:"fileSystemInfo,omitempty"`
	Folder               *Folder         `json:"folder,omitempty"`
	// Audio
	// Image
	// Location
	// OpenWith
	// Photo
	// SpecialFolder
	// Video
	// Permissions
	// Versions
	// Children
	// Thumbnails
}

type ItemUpdateBody struct {
	Name            string         `json:"name,omitempty"`
	ParentReference *ItemReference `json:"parentReference,omitempty"`
}

type ItemCreateBody struct {
	Name   string `json:"name,omitempty"`
	Folder Folder `json:"folder,omitempty"`
}

type ItemCopyBody struct {
	Name            string         `json:"name,omitempty"`
	ParentReference *ItemReference `json:"parentReference,omitempty"`
}

type ItemCollectionPage struct {
	Value    []*Item `json:"value"`
	NextLink string  `json:"@odata.nextLink"`
}

type DeltaCollectionPage struct {
	Value    []*Item `json:"value"`
	NextLink string  `json:"@odata.nextLink"`
	Token    string  `json:"@delta.token"`
}

type CreateSessionBody struct {
	Item ChunkedUploadSessionDescriptor `json:"item"`
}

type AsyncOperationStatus struct {
	Operation          string  `json:"operation"`
	PercentageComplete float64 `json:"percentageComplete"`
	Status             string  `json:"status"`
}

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
