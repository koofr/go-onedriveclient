package onedriveclient

import (
	"bytes"
	"fmt"
	"github.com/koofr/go-httpclient"
	"github.com/koofr/go-ioutils"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OneDrive", func() {
	var client *OneDrive

	auth := &OneDriveAuth{
		ClientId:     os.Getenv("ONEDRIVE_CLIENT_ID"),
		ClientSecret: os.Getenv("ONEDRIVE_CLIENT_SECRET"),
		RedirectUri:  os.Getenv("ONEDRIVE_REDIRECT_URI"),
		AccessToken:  os.Getenv("ONEDRIVE_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("ONEDRIVE_REFRESH_TOKEN"),
	}

	if auth.ClientId == "" || auth.ClientSecret == "" || auth.RedirectUri == "" || auth.AccessToken == "" || auth.RefreshToken == "" || os.Getenv("ONEDRIVE_EXPIRES_AT") == "" {
		fmt.Println("ONEDRIVE_CLIENT_ID, ONEDRIVE_CLIENT_SECRET, ONEDRIVE_ACCESS_TOKEN, ONEDRIVE_REFRESH_TOKEN, ONEDRIVE_EXPIRES_AT env variable missing")
		return
	}

	exp, _ := strconv.ParseInt(os.Getenv("ONEDRIVE_EXPIRES_AT"), 10, 0)
	auth.ExpiresAt = time.Unix(0, exp*1000000)

	BeforeEach(func() {
		client = NewOneDrive(auth)

		children := &struct {
			Value []*Item
		}{}

		req := &httpclient.RequestData{
			Method:         "GET",
			Path:           "/drive/root/children",
			ExpectedStatus: []int{http.StatusOK},
			RespEncoding:   httpclient.EncodingJSON,
			RespValue:      &children,
		}

		_, err := client.Request(req)
		Expect(err).NotTo(HaveOccurred())

		for _, item := range children.Value {
			req := &httpclient.RequestData{
				Method:         "DELETE",
				Path:           "/drive/root:/" + item.Name,
				ExpectedStatus: []int{http.StatusNoContent},
				RespConsume:    true,
			}

			_, err := client.Request(req)
			Expect(err).NotTo(HaveOccurred())
		}

		_, err = client.Upload("file.txt", NameConflictBehaviorReplace, bytes.NewBufferString("12345"), 5)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Info", func() {
		It("should get file info", func() {
			item, err := client.Info("file.txt")
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Size).To(Equal(int64(5)))
		})
	})

	Describe("Download", func() {
		It("should download file", func() {
			item, err := client.Download("file.txt", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Size).To(Equal(int64(5)))

			data, _ := ioutil.ReadAll(item.Reader)
			item.Reader.Close()

			Expect(string(data)).To(Equal("12345"))
		})

		It("should download file range", func() {
			item, err := client.Download("file.txt", &ioutils.FileSpan{2, 3})
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Size).To(Equal(int64(2)))

			data, _ := ioutil.ReadAll(item.Reader)
			item.Reader.Close()

			Expect(string(data)).To(Equal("34"))
		})
	})

	Describe("Upload", func() {
		It("should upload file with nonexisting name", func() {
			client.MaxFragmentSize = 3
			data := bytes.NewBufferString("12345")
			item, err := client.Upload("/new-file.txt", NameConflictBehaviorRename, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("new-file.txt"))
		})

		It("should upload file with existing name", func() {
			data := bytes.NewBufferString("12345")
			item, err := client.Upload("file.txt", NameConflictBehaviorRename, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("file 1.txt"))

			item, err = client.Info("file 1.txt")
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should overwrite existing file", func() {
			data := bytes.NewBufferString("12345")
			item, err := client.Upload("file.txt", NameConflictBehaviorReplace, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("file.txt"))

			item, err = client.Info("file.txt")
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should not autorename", func() {
			data := bytes.NewBufferString("12345")
			_, err := client.Upload("file.txt", NameConflictBehaviorFail, data, 5)
			Expect(err).To(HaveOccurred())
		})
	})

})
