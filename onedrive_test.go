package onedriveclient

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/koofr/go-ioutils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OneDrive", func() {
	var client *OneDrive
	var fileItem *Item

	driveID := os.Getenv("ONEDRIVE_DRIVE_ID")
	isGraph := driveID != ""

	auth := &OneDriveAuth{
		ClientId:     os.Getenv("ONEDRIVE_CLIENT_ID"),
		ClientSecret: os.Getenv("ONEDRIVE_CLIENT_SECRET"),
		RedirectUri:  os.Getenv("ONEDRIVE_REDIRECT_URI"),
		AccessToken:  os.Getenv("ONEDRIVE_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("ONEDRIVE_REFRESH_TOKEN"),
		IsGraph:      isGraph,
	}

	if auth.ClientId == "" || auth.ClientSecret == "" || auth.RedirectUri == "" || auth.AccessToken == "" || auth.RefreshToken == "" || os.Getenv("ONEDRIVE_EXPIRES_AT") == "" {
		fmt.Println("ONEDRIVE_CLIENT_ID, ONEDRIVE_CLIENT_SECRET, ONEDRIVE_ACCESS_TOKEN, ONEDRIVE_REFRESH_TOKEN, ONEDRIVE_EXPIRES_AT env variable missing")
		return
	}

	exp, _ := strconv.ParseInt(os.Getenv("ONEDRIVE_EXPIRES_AT"), 10, 0)
	auth.ExpiresAt = time.Unix(0, exp*1000000)

	BeforeEach(func() {
		if isGraph {
			client = NewOneDriveGraph(auth, driveID)
		} else {
			client = NewOneDrive(auth)
		}

		children, err := client.ItemsChildren(context.Background(), AddressRoot, "")
		Expect(err).NotTo(HaveOccurred())

		for _, item := range children.Value {
			err = client.ItemsDelete(context.Background(), AddressId(item.Id))
			Expect(err).NotTo(HaveOccurred())
		}

		fileItem, err = client.ItemsUpload(context.Background(), AddressRoot, "file.txt", NameConflictBehaviorReplace, bytes.NewBufferString("12345"), 5)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Drive", func() {
		It("should get default drive", func() {
			drive, err := client.Drive(context.Background())
			Expect(err).NotTo(HaveOccurred())

			if isGraph {
				if drive.DriveType == "business" {
					Expect(drive.DriveType).To(Equal("business"))
				} else {
					Expect(drive.DriveType).To(Equal("personal"))
				}
			} else {
				Expect(drive.DriveType).To(Equal("personal"))
			}
		})
	})

	Describe("ItemsGet", func() {
		It("should get item info by path", func() {
			item, err := client.ItemsGet(context.Background(), AddressPath("/file.txt"))
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should get item info by id", func() {
			item, err := client.ItemsGet(context.Background(), AddressId(fileItem.Id))
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should not get deleted item", func() {
			err := client.ItemsDelete(context.Background(), AddressId(fileItem.Id))
			Expect(err).NotTo(HaveOccurred())

			_, err = client.ItemsGet(context.Background(), AddressPath("/file.txt"))
			Expect(err).To(HaveOccurred())

			ode, ok := IsOneDriveError(err)
			Expect(ok).To(BeTrue())
			Expect(ode.Err.Code).To(Equal(ErrorCodeItemNotFound))
		})
	})

	Describe("ItemsUpdate", func() {
		It("should rename item", func() {
			itemUpdate := &ItemUpdateBody{
				Name: "renamed.txt",
			}

			item, err := client.ItemsUpdate(context.Background(), AddressId(fileItem.Id), itemUpdate)
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Name).To(Equal("renamed.txt"))

			_, err = client.ItemsGet(context.Background(), AddressPath("/renamed.txt"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not rename item if it already exists", func() {
			_, err := client.ItemsUpload(context.Background(), AddressRoot, "existing.txt", NameConflictBehaviorReplace, bytes.NewBufferString("123456"), 6)
			Expect(err).NotTo(HaveOccurred())

			itemUpdate := &ItemUpdateBody{
				Name: "existing.txt",
			}

			_, err = client.ItemsUpdate(context.Background(), AddressId(fileItem.Id), itemUpdate)
			Expect(err).To(HaveOccurred())

			ode, ok := IsOneDriveError(err)
			Expect(ok).To(BeTrue())
			Expect(ode.Err.Code).To(Equal(ErrorCodeNameAlreadyExists))
		})

		It("should move item", func() {
			dirItem, err := client.ItemsCreate(context.Background(), AddressRoot, &ItemCreateBody{Name: "dir"})
			Expect(err).NotTo(HaveOccurred())

			itemUpdate := &ItemUpdateBody{
				ParentReference: &ItemReference{
					Id: dirItem.Id,
				},
			}

			item, err := client.ItemsUpdate(context.Background(), AddressId(fileItem.Id), itemUpdate)
			Expect(err).NotTo(HaveOccurred())

			Expect(item.ParentReference.Id).To(Equal(dirItem.Id))

			_, err = client.ItemsGet(context.Background(), AddressPath("/dir/file.txt"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ItemsDelete", func() {
		It("should delete", func() {
			err := client.ItemsDelete(context.Background(), AddressId(fileItem.Id))
			Expect(err).NotTo(HaveOccurred())

			_, err = client.ItemsGet(context.Background(), AddressPath("/file.txt"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ItemsCreate", func() {
		It("should create a new folder", func() {
			itemCreate := &ItemCreateBody{
				Name: "new folder",
			}

			item, err := client.ItemsCreate(context.Background(), AddressRoot, itemCreate)
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Name).To(Equal("new folder"))

			_, err = client.ItemsGet(context.Background(), AddressPath("/new folder"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ItemsChildren", func() {
		It("should get children", func() {
			children, err := client.ItemsChildren(context.Background(), AddressRoot, "")
			Expect(err).NotTo(HaveOccurred())

			Expect(children.Value).To(HaveLen(1))

			Expect(children.Value[0].Name).To(Equal("file.txt"))
		})
	})

	Describe("ItemsCopy", func() {
		It("should create a file copy", func() {
			monitorUrl, err := client.ItemsCopy(context.Background(), AddressId(fileItem.Id), &ItemCopyBody{Name: "file copy.txt"})
			Expect(err).NotTo(HaveOccurred())

			item, err := client.ItemsCopyAwait(context.Background(), monitorUrl)

			if isGraph {
				Expect(err).To(Equal(ErrCompletedNoItem))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(item.Name).To(Equal("file copy.txt"))
			}

			_, err = client.ItemsGet(context.Background(), AddressPath("/file copy.txt"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a file copy into path", func() {
			destItem, err := client.ItemsCreate(context.Background(), AddressRoot, &ItemCreateBody{Name: "dest"})
			Expect(err).NotTo(HaveOccurred())

			monitorUrl, err := client.ItemsCopy(context.Background(), AddressId(fileItem.Id), &ItemCopyBody{
				Name: "file copy.txt",
				ParentReference: &ItemReference{
					Id: destItem.Id,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			item, err := client.ItemsCopyAwait(context.Background(), monitorUrl)

			if isGraph {
				Expect(err).To(Equal(ErrCompletedNoItem))
			} else {
				Expect(err).NotTo(HaveOccurred())

				Expect(item.Name).To(Equal("file copy.txt"))
			}

			_, err = client.ItemsGet(context.Background(), AddressPath("/dest/file copy.txt"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create a dir copy", func() {
			dirItem, err := client.ItemsCreate(context.Background(), AddressRoot, &ItemCreateBody{Name: "dir"})
			Expect(err).NotTo(HaveOccurred())

			_, err = client.ItemsUpload(context.Background(), AddressId(dirItem.Id), "file.txt", NameConflictBehaviorReplace, bytes.NewBufferString("123456"), 6)
			Expect(err).NotTo(HaveOccurred())

			monitorUrl, err := client.ItemsCopy(context.Background(), AddressId(dirItem.Id), &ItemCopyBody{Name: "dir copy"})
			Expect(err).NotTo(HaveOccurred())

			item, err := client.ItemsCopyAwait(context.Background(), monitorUrl)
			if isGraph {
				Expect(err).To(Equal(ErrCompletedNoItem))
			} else {
				Expect(err).NotTo(HaveOccurred())

				Expect(item.Name).To(Equal("dir copy"))
			}

			_, err = client.ItemsGet(context.Background(), AddressPath("/dir copy"))
			Expect(err).NotTo(HaveOccurred())

			_, err = client.ItemsGet(context.Background(), AddressPath("/dir copy/file.txt"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ItemsDelta", func() {
		It("should get items delta", func() {
			firstDelta, err := client.ItemsDelta(context.Background(), AddressRoot, "", "")
			Expect(err).NotTo(HaveOccurred())

			if isGraph {
				Expect(len(firstDelta.Value)).To(BeNumerically(">", 0))
			} else {
				Expect(len(firstDelta.Value)).To(BeNumerically(">=", 0))
				Expect(firstDelta.Value[0].Name).To(Equal("root"))
				Expect(firstDelta.Value[1].Name).To(Equal("file.txt"))

				delta, err := client.ItemsDelta(context.Background(), AddressRoot, firstDelta.NextLink, "")
				Expect(err).NotTo(HaveOccurred())

				Expect(len(firstDelta.Value)).To(BeNumerically(">=", 1))
				Expect(delta.Value[0].Name).To(Equal("root"))

				delta, err = client.ItemsDelta(context.Background(), AddressRoot, "", firstDelta.Token)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(firstDelta.Value)).To(BeNumerically(">=", 1))
				Expect(delta.Value[0].Name).To(Equal("root"))

				err = client.ItemsDelete(context.Background(), AddressId(fileItem.Id))
				Expect(err).NotTo(HaveOccurred())

				delta, err = client.ItemsDelta(context.Background(), AddressRoot, "", firstDelta.Token)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(firstDelta.Value)).To(BeNumerically(">=", 2))
				Expect(delta.Value[0].Name).To(Equal("root"))
				Expect(delta.Value[1].Name).To(Equal("file.txt"))
				Expect(delta.Value[1].Deleted).NotTo(BeNil())
			}
		})
	})

	Describe("ItemsContent", func() {
		It("should get content", func() {
			reader, size, err := client.ItemsContent(context.Background(), AddressId(fileItem.Id), nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(int64(5)))

			data, _ := ioutil.ReadAll(reader)
			reader.Close()

			Expect(string(data)).To(Equal("12345"))
		})

		It("should get content range", func() {
			reader, size, err := client.ItemsContent(context.Background(), AddressId(fileItem.Id), &ioutils.FileSpan{Start: 2, End: 3})
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(int64(2)))

			data, _ := ioutil.ReadAll(reader)
			reader.Close()

			Expect(string(data)).To(Equal("34"))
		})
	})

	Describe("ItemsUpload", func() {
		It("should upload file with nonexisting name", func() {
			client.MaxFragmentSize = 3
			data := bytes.NewBufferString("12345")
			item, err := client.ItemsUpload(context.Background(), AddressRoot, "new-file.txt", NameConflictBehaviorRename, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("new-file.txt"))
		})

		It("should upload file with nonexisting name using path", func() {
			client.MaxFragmentSize = 3
			data := bytes.NewBufferString("12345")
			item, err := client.ItemsUpload(context.Background(), AddressPath("/new-file.txt"), "new-file.txt", NameConflictBehaviorRename, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("new-file.txt"))
		})

		It("should upload file with existing name", func() {
			data := bytes.NewBufferString("12345")
			item, err := client.ItemsUpload(context.Background(), AddressRoot, "file.txt", NameConflictBehaviorRename, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("file 1.txt"))

			item, err = client.ItemsGet(context.Background(), AddressPath("/file 1.txt"))
			Expect(err).NotTo(HaveOccurred())

			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should overwrite existing file", func() {
			data := bytes.NewBufferString("12345")
			item, err := client.ItemsUpload(context.Background(), AddressRoot, "file.txt", NameConflictBehaviorReplace, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("file.txt"))

			item, err = client.ItemsGet(context.Background(), AddressPath("/file.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should overwrite existing file in folder", func() {
			dirItem, err := client.ItemsCreate(context.Background(), AddressRoot, &ItemCreateBody{Name: "dir"})
			Expect(err).NotTo(HaveOccurred())

			data := bytes.NewBufferString("12345")
			item, err := client.ItemsUpload(context.Background(), AddressId(dirItem.Id), "file.txt", NameConflictBehaviorReplace, data, 5)
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Name).To(Equal("file.txt"))

			item, err = client.ItemsGet(context.Background(), AddressPath("/dir/file.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Size).To(Equal(int64(5)))
		})

		It("should not autorename", func() {
			data := bytes.NewBufferString("12345")
			_, err := client.ItemsUpload(context.Background(), AddressRoot, "file.txt", NameConflictBehaviorFail, data, 5)
			Expect(err).To(HaveOccurred())
		})
	})

})
