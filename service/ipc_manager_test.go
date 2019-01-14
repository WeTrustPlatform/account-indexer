package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextIPC(t *testing.T) {
	ipcList := []string{"ipc1", "ipc2", "ipc3"}
	assert.Equal(t, "ipc2", NextIPC("ipc1", ipcList))
	assert.Equal(t, "ipc3", NextIPC("ipc2", ipcList))
	assert.Equal(t, "ipc1", NextIPC("ipc3", ipcList))
}

func TestSetIPC(t *testing.T) {
	im := GetIpcManager()
	err := im.SetIPC(nil)
	assert.NotNil(t, err)
	err = im.SetIPC([]string{"ipc1", "ipc1"})
	assert.NotNil(t, err)
	err = im.SetIPC([]string{"ipc1", ""})
	assert.NotNil(t, err)
	err = im.SetIPC([]string{"ipc1", "ipc2"})
	assert.Nil(t, err)
	assert.Equal(t, "ipc1", im.curIPC)
}

type IpcSubscriberImpl struct {
	count int
}

func (is *IpcSubscriberImpl) IpcUpdated(ipc string) {
	is.count++
}

func TestChangeIPC(t *testing.T) {
	im := GetIpcManager()
	err := im.SetIPC([]string{"ipc1", "ipc2"})
	assert.Nil(t, err)
	sub := IpcSubscriberImpl{}
	assert.Equal(t, 0, sub.count)
	var tmp IpcSubscriber
	tmp = &sub
	im.Subscribe(&tmp)
	im.ChangeIPC()
	assert.Equal(t, 1, sub.count)
	im.ChangeIPC()
	assert.Equal(t, 2, sub.count)
}
