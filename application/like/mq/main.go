package main

import (
	"context"
	"flag"

	"beyond/application/like/mq/internal/config"
	"beyond/application/like/mq/internal/logic"
	"beyond/application/like/mq/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/like.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	svcCtx := svc.NewServiceContext(c)
	ctx := context.Background()
	//创建一个新的服务组，并在程序结束时停止服务组中的所有服务。
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	//Consumers返回一个服务切片，遍历所有消息队列消费者，并将它们添加到服务组中。
	for _, mq := range logic.Consumers(ctx, svcCtx) {
		serviceGroup.Add(mq)
	}
	//启动服务组中的所有服务
	serviceGroup.Start()
}
