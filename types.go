package onedriveclient

type RefreshResp struct {
	ExpiresIn   int64  `json:"expires_in"`
	AccessToken string `json:"access_token"`
}

type NodeInfo struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	Type        string `json:"type"`
	UpdatedTime string `json:"updated_time"`
	Source      string `json:"source,omitempty"`
}

type NodeFiles struct {
	Data []NodeInfo `json:"data"`
}
