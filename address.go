package onedriveclient

import (
	"path"
)

const (
	AddressTypeId   = 0
	AddressTypePath = 1
)

type Address struct {
	Address string
	Type    int
}

func (a Address) Subpath(path string) Address {
	return Address{
		Address: a.Address + path,
		Type:    a.Type,
	}
}

func (a Address) String(driveId string) string {
	if driveId == "" {
		return "/drive" + a.Address
	}

	return "/drives/" + driveId + a.Address
}

var AddressRoot = Address{
	Address: "/items/root",
	Type:    AddressTypeId,
}

func AddressId(id string) Address {
	return Address{
		Address: "/items/" + id,
		Type:    AddressTypeId,
	}
}

func AddressPath(pth string) Address {
	return Address{
		Address: "/root:" + NormalizePath(pth) + ":",
		Type:    AddressTypePath,
	}
}

func NormalizePath(pth string) string {
	return path.Clean("/" + pth)
}
