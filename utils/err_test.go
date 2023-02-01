package utils

import (
	"context"
	"sync"
	"testing"

	"github.com/canbefree/pkgs/common"
	"github.com/stretchr/testify/assert"
)

func samplePanicGo() error {
	var err error
	s := sync.WaitGroup{}
	s.Add(1)
	l := common.NewDefaultLogger()
	go GO(l, func(ctx context.Context) error {
		defer func() {
			s.Done()
		}()
		panic("error")
	})(context.TODO())

	s.Wait()
	return err
}

func TestGO(t *testing.T) {
	err := samplePanicGo()
	assert.Nil(t, err)
}
