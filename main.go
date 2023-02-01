package main

import (
	"context"
	"time"

	"github.com/canbefree/pkgs/common"
	"github.com/canbefree/pkgs/utils"
)

func main() {
	l := common.NewDefaultLogger()
	go utils.GO(l, func(ctx context.Context) error {
		panic("not implemented")
	})(context.TODO())
	time.Sleep(time.Second)
}
