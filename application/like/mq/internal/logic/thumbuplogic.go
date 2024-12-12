package logic

import (
	"context"
	"fmt"

	"beyond/application/like/mq/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

type ThumbupLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewThumbupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ThumbupLogic {
	return &ThumbupLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ThumbupLogic) Consume(key, val string) error {
	//这里消费逻辑是直接打印，后面再补充
	fmt.Printf("get key: %s val: %s\n", key, val)
	return nil
}

func Consumers(ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	//返回一个服务切片，用于启动消费者服务。
	return []service.Service{
		kq.MustNewQueue(svcCtx.Config.KqConsumerConf, NewThumbupLogic(ctx, svcCtx)),
	}
}
