package utils

import (
	"context"
	"log"

	"github.com/canbefree/pkgs/common"
)

func PanicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

// GO usage go Go(...)(ctx)
func GO(l common.LoggerIFace, fn func(ctx context.Context) error) func(context.Context) {
	return func(ctx context.Context) {
		defer func() {
			if x := recover(); x != nil {
				if err, ok := x.(error); ok {
					l.Fatalf(ctx, "recover from panic, err: %v", err)
				} else {
					l.Fatalf(ctx, "recover from panic, panic: %v", x)
				}
			}
		}()
		if err := fn(ctx); err != nil {
			log.Fatalf("GO err:%v", err)
		}
	}
}
