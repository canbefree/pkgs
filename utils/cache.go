package utils

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	pre_proto "github.com/golang/protobuf/proto"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// 旁路缓存 -> 获取缓存结果 -> 命中  	--> 返回结果
// 						-> 未命中 	--> 查询 -> 结果为空 -> 缓存空结果
// 											  不为空 -> 缓存结果
// 						-> 异常 	-> 返回异常

type GrpcSession struct {
	Request proto.Message
	Resp    proto.Message
	Err     error
}

func NewGrpcSession(req, resp proto.Message, err error) (*GrpcSession, error) {
	if req == nil {
		return nil, fmt.Errorf("grpc: Request nil ")
	}
	if req == nil {
		return nil, fmt.Errorf("grpc: Response nil ")
	}
	return &GrpcSession{
		Request: req,
		Resp:    resp,
		Err:     err,
	}, nil
}

type midCache struct {
	Resp []byte
	Err  struct {
		Code codes.Code
		MSG  string
	}
}

func (r *GrpcSession) Bytes() ([]byte, error) {
	m := &midCache{}

	if r.Resp == nil {
		return nil, nil
	}

	var messageV2 = pre_proto.MessageV2(r.Resp)
	if !messageV2.ProtoReflect().IsValid() {
		return nil, fmt.Errorf("grpc: unexpected message type")
	}
	// GRPC response 处理
	v, err := proto.Marshal(messageV2)
	if err != nil {
		return nil, err
	}
	m.Resp = v

	// GRPC err 处理
	if r.Err != nil {
		m.Err.Code = status.Code(r.Err)
		m.Err.MSG = r.Err.Error()
	}

	s, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *GrpcSession) Loads(data []byte) error {
	m := &midCache{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	typ := reflect.TypeOf(r.Resp)
	val := reflect.ValueOf(r.Resp)

	x1 := reflect.New(typ)
	x2 := x1.Elem()
	x2.Set(val)
	x := x2.Interface()
	res, ok := x.(proto.Message)
	if !ok {
		return fmt.Errorf("error creating")
	}

	if err := proto.Unmarshal(m.Resp, res); err != nil {
		return err
	}

	r.Resp = res
	r.Err = status.Error(m.Err.Code, m.Err.MSG)
	return nil
}

func (r *GrpcSession) Response() (proto.Message, error) {
	return r.Resp, r.Err
}

//go:generate mockery --name=CacheEngine --outpkg=utils_mock
type CacheEngine interface {
	// 从缓存读取字符串内容
	GetBytesFromCache(ctx context.Context, key []byte) ([]byte, error)

	// 设置缓存
	SetBytesInfoCache(ctx context.Context, key []byte, cache interface{}, expired int) error
}

type GRPCCache struct {
	CacheEngine                                      // 缓存引擎
	keyGenerator func(proto.Message) ([]byte, error) //

	psession *GrpcSession
	session  *GrpcSession

	directFn func() (interface{}, error) // 这个就是一个普通的grpc请求

	expired int
	group   *singleflight.Group

	E []error // 缓存错误列表 错误结果也会缓存
}

func NewGrpcCache(cahceEngine CacheEngine, directFn func() (interface{}, error), s *GrpcSession, group *singleflight.Group, expired int) *GRPCCache {
	return &GRPCCache{
		CacheEngine: cahceEngine,
		psession:    s,
		session:     s,
		group:       group,
		expired:     expired,
		directFn:    directFn,
		keyGenerator: func(req proto.Message) ([]byte, error) {
			b, err := proto.Marshal(req)
			bt := md5.Sum(b)
			return bt[:], err
		},
	}
}

func (g *GRPCCache) AddE(err ...error) {
	g.E = err
}

func (g *GRPCCache) Get(ctx context.Context) (proto.Message, error) {
	key, err := g.keyGenerator(g.psession.Request)
	if err != nil {
		return nil, err
	}

	var wrapFn = func() (interface{}, error) {
		bytes, err := g.CacheEngine.GetBytesFromCache(ctx, key)
		if err != nil {
			iresp, err := g.directFn()
			var resp proto.Message
			if !reflect.ValueOf(iresp).IsNil() {
				var ok bool
				resp, ok = iresp.(proto.Message)
				if !ok {
					return nil, fmt.Errorf("resp not a proto.Message")
				}
			} else {
				resp = g.session.Resp
			}

			if err != nil && len(g.E) > 0 {
				for _, e := range g.E {
					if !errors.Is(err, e) {
						return NewGrpcSession(g.psession.Request, resp, err)
					}
				}
			}

			g.session, err = NewGrpcSession(g.psession.Request, resp, err)
			if err != nil {
				return nil, err
			}

			bytes, err := g.session.Bytes()
			if err != nil {
				return nil, err
			}

			if err := g.SetBytesInfoCache(ctx, key, bytes, int(g.expired)); err != nil {
				return nil, err
			}
			return g.session, nil
		}

		if err := g.session.Loads(bytes); err != nil {
			panic(err)
		}
		return g.session, nil
	}

	iresp, err, _ := g.group.Do(string(key), wrapFn)
	if err != nil {
		return nil, err
	}

	if iresp != nil {
		session := iresp.(*GrpcSession)
		return session.Response()
	}
	return nil, nil
}
