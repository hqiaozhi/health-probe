package tcp

import (
	"context"
	"net"
	"time"
)

type TCP interface {
	Probe(ctx context.Context, in *Request) (*Response, error)
}

type Probe struct {
}

type Request struct {
	Target  string
	Port    string
	Timeout int64
}

type Response struct {
	Success  bool
	Duration int64
}

func New() TCP {
	return &Probe{}
}

func (l *Probe) Probe(ctx context.Context, in *Request) (*Response, error) {
	start := time.Now()
	timeout := time.Duration(in.Timeout) * time.Second
	if in.Timeout <= 0 {
		timeout = 5 * time.Second // 默认超时5秒
	}

	Address := net.JoinHostPort(in.Target, in.Port)
	// 尝试建立TCP连接
	conn, err := net.DialTimeout("tcp", Address, timeout)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return &Response{
			Success:  false,
			Duration: duration,
		}, err
	}
	defer conn.Close()
	// 连接成功
	return &Response{
		Success:  true,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}
