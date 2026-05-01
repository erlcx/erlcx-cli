//go:build windows

package auth

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

type nativeCredential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWrite  = advapi32.NewProc("CredWriteW")
	procCredRead   = advapi32.NewProc("CredReadW")
	procCredDelete = advapi32.NewProc("CredDeleteW")
	procCredFree   = advapi32.NewProc("CredFree")
)

func (store WindowsCredentialStore) Save(credential StoredCredential) error {
	data, err := encodeCredential(credential)
	if err != nil {
		return err
	}

	target, err := syscall.UTF16PtrFromString(store.target())
	if err != nil {
		return err
	}
	user, err := syscall.UTF16PtrFromString("roblox")
	if err != nil {
		return err
	}

	native := nativeCredential{
		Type:               credTypeGeneric,
		TargetName:         target,
		CredentialBlobSize: uint32(len(data)),
		CredentialBlob:     &data[0],
		Persist:            credPersistLocalMachine,
		UserName:           user,
	}

	ret, _, callErr := procCredWrite.Call(uintptr(unsafe.Pointer(&native)), 0)
	if ret == 0 {
		return fmt.Errorf("save Windows credential: %w", callErr)
	}
	return nil
}

func (store WindowsCredentialStore) Load() (StoredCredential, error) {
	target, err := syscall.UTF16PtrFromString(store.target())
	if err != nil {
		return StoredCredential{}, err
	}

	var native *nativeCredential
	ret, _, callErr := procCredRead.Call(
		uintptr(unsafe.Pointer(target)),
		uintptr(credTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&native)),
	)
	if ret == 0 {
		if errors.Is(callErr, syscall.ERROR_NOT_FOUND) {
			return StoredCredential{}, ErrNotLoggedIn
		}
		return StoredCredential{}, fmt.Errorf("read Windows credential: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(native)))

	data := unsafe.Slice(native.CredentialBlob, native.CredentialBlobSize)
	return decodeCredential(data)
}

func (store WindowsCredentialStore) Delete() error {
	target, err := syscall.UTF16PtrFromString(store.target())
	if err != nil {
		return err
	}

	ret, _, callErr := procCredDelete.Call(
		uintptr(unsafe.Pointer(target)),
		uintptr(credTypeGeneric),
		0,
	)
	if ret == 0 {
		if errors.Is(callErr, syscall.ERROR_NOT_FOUND) {
			return ErrNotLoggedIn
		}
		return fmt.Errorf("delete Windows credential: %w", callErr)
	}
	return nil
}
