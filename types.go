package onedriveclient

import (
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

type GraphChunkedUploadSessionDescriptor struct {
	NameConflictBehavior string `json:"@microsoft.graph.conflictBehavior"`
	Name                 string `json:"name,omitempty"`
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

type Timestamp time.Time

const timestampFormat = `"` + time.RFC3339 + `"`

func (t *Timestamp) MarshalJSON() (out []byte, err error) {
	timeString := (*time.Time)(t).UTC().Format(timestampFormat)
	return []byte(timeString), nil
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	newT, err := time.Parse(timestampFormat, string(data))
	if err != nil {
		return err
	}
	*t = Timestamp(newT)
	return nil
}

type FileSystemInfo struct {
	CreatedDateTime      Timestamp `json:"createdDateTime"`
	LastModifiedDateTime Timestamp `json:"lastModifiedDateTime"`
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
	Name            string          `json:"name,omitempty"`
	ParentReference *ItemReference  `json:"parentReference,omitempty"`
	FileSystemInfo  *FileSystemInfo `json:"fileSystemInfo,omitempty"`
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

	DeltaLink string `json:"@odata.deltaLink"`
}

type BaseCreateSessionBody interface {
	GetName() string
}

type CreateSessionBody struct {
	Item ChunkedUploadSessionDescriptor `json:"item"`
}

func (b *CreateSessionBody) GetName() string {
	return b.Item.Name
}

type GraphCreateSessionBody struct {
	Item GraphChunkedUploadSessionDescriptor `json:"item"`
}

func (b *GraphCreateSessionBody) GetName() string {
	return b.Item.Name
}

type AsyncOperationStatus struct {
	Operation          string  `json:"operation"`
	PercentageComplete float64 `json:"percentageComplete"`
	Status             string  `json:"status"`
}
