package handler

import (
	"context"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/opentracing/opentracing-go"

	"github.com/govinda-attal/istio-me/pkg/trials"
)

type greeterImpl struct {
	timer trials.TimerClient
}

func NewGreeterSrv(timer trials.TimerClient) *greeterImpl {
	return &greeterImpl{timer}
}

func (g *greeterImpl) Hello(ctx context.Context, in *trials.HelloRq) (*trials.HelloRs, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "Hello")
	defer span.Finish()
	span.LogEventWithPayload("request", in)
	rs, err := g.timer.Time(ctx, &empty.Empty{})
	if err != nil {
		return nil, err
	}
	return &trials.HelloRs{
		Msg: "hello " + in.GetName() + "@ " + rs.Msg,
	}, nil
}

type timerImpl struct{}

func NewTimerSrv() *timerImpl {
	return &timerImpl{}
}

func (t *timerImpl) Time(ctx context.Context, _ *empty.Empty) (*trials.TimeRs, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "recordedTime")
	defer span.Finish()
	ts := time.Now().Format(time.RFC3339)
	span.LogKV("time", ts)
	return &trials.TimeRs{
		Msg: ts,
	}, nil
}
