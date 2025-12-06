package http

import (
	"context"
	"net/http"
	"time"
)

type HTTP interface {
	Probe(ctx context.Context, in *Request) (*Response, error)
}

type Probe struct {
}

type Request struct {
	Tls     bool
	Target  string
	Method  string
	Path    string
	Timeout int64
}

type Response struct {
	Success  bool
	Code     int
	Duration int64
}

func New() HTTP {
	return &Probe{}
}

func (l *Probe) Probe(ctx context.Context, in *Request) (*Response, error) {
	start := time.Now()
	timeout := time.Duration(in.Timeout) * time.Second
	if in.Timeout <= 0 {
		timeout = 5 * time.Second // 默认超时5秒
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: timeout,
	}

	// 构建URL
	if in.Tls {
		in.Target = "https://" + in.Target
	} else {
		in.Target = "http://" + in.Target
	}

	// 发送HEAD请求(减少数据传输)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, in.Target, nil)
	if err != nil {
		return &Response{
			Success:  false,
			Duration: time.Since(start).Milliseconds(),
		}, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)

	if err != nil {
		return &Response{
			Success:  false,
			Duration: time.Since(start).Milliseconds(),
		}, nil
	}
	defer resp.Body.Close()
	return &Response{
		Success:  true,
		Code:     resp.StatusCode,
		Duration: time.Since(start).Milliseconds(),
	}, nil
}
