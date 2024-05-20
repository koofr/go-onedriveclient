package onedriveclient

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGoOnedriveclient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GoOnedriveclient Suite")
}
