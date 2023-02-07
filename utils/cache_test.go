package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	utils_mock "github.com/canbefree/pkgs/utils/mocks"
	"github.com/canbefree/pkgs/utils/pb_demo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func getDemo(ctx context.Context, req *pb_demo.CreateDemoRequest) (*pb_demo.Demo, error) {
	if req.DemoId == "" {
		return nil, errors.New("request err")
	}
	return &pb_demo.Demo{
		Name: "hello,world",
	}, nil
}

func Test_GRPCCache_Get(t *testing.T) {
	ctx := context.TODO()
	var cacheEngine = utils_mock.NewCacheEngine(t)
	respStr, _ := proto.Marshal(&pb_demo.Demo{
		Name: "haha",
	})
	msg, _ := json.Marshal(&midCache{
		Resp: respStr,
		Err: struct {
			Code codes.Code
			MSG  string
		}{
			Code: 0,
			MSG:  "",
		},
	})
	cacheEngine.On("GetBytesFromCache", ctx, mock.Anything).Return(msg, nil).Times(1)
	// cacheEngine.On("SetBytesInfoCache", ctx, mock.Anything, mock.Anything, 10).Return(nil).Times(1)
	req := &pb_demo.CreateDemoRequest{}
	g := &singleflight.Group{}
	grpcCache := NewGrpcCache(cacheEngine, func() (interface{}, error) {
		return getDemo(ctx, req)
	}, NewGrpcSession(req, nil, nil), g, 10)
	iresp, err := grpcCache.Get(ctx)
	resp, ok := iresp.(*pb_demo.Demo)
	assert.Equal(t, true, ok)
	if ok {
		assert.Equal(t, "haha", resp.Name)
	}
	assert.Equal(t, err, nil)
}

func Test_GRPCCache_Get_Miss(t *testing.T) {
	ctx := context.TODO()
	var cacheEngine = utils_mock.NewCacheEngine(t)
	cacheEngine.On("GetBytesFromCache", ctx, mock.Anything).Return(nil, fmt.Errorf("cache miss")).Times(1)
	cacheEngine.On("SetBytesInfoCache", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(1)
	req := &pb_demo.CreateDemoRequest{}
	g := &singleflight.Group{}
	grpcCache := NewGrpcCache(cacheEngine, func() (interface{}, error) {
		return getDemo(ctx, req)
	}, NewGrpcSession(req, nil, nil), g, 10)
	iresp, err := grpcCache.Get(ctx)
	assert.Equal(t, nil, iresp)
	assert.Equal(t, err, fmt.Errorf("request err"))
}

func Test_GRPCCache_Get_E(t *testing.T) {
	ctx := context.TODO()
	var cacheEngine = utils_mock.NewCacheEngine(t)
	cacheEngine.On("GetBytesFromCache", ctx, mock.Anything).Return(nil, fmt.Errorf("cache miss")).Times(1)
	// cacheEngine.On("SetBytesInfoCache", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(0)
	req := &pb_demo.CreateDemoRequest{}
	g := &singleflight.Group{}
	grpcCache := NewGrpcCache(cacheEngine, func() (interface{}, error) {
		return getDemo(ctx, req)
	}, NewGrpcSession(req, nil, nil), g, 10)

	grpcCache.AddE(errors.New("request err"))

	iresp, err := grpcCache.Get(ctx)
	assert.Equal(t, nil, iresp)
	assert.Equal(t, err, fmt.Errorf("request err"))
}
