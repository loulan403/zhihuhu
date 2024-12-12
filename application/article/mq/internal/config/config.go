package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	service.ServiceConf

	KqConsumerConf        kq.KqConf //点赞计数的topic
	ArticleKqConsumerConf kq.KqConf //消费文章变更的topic
	Datasource            string
	BizRedis              redis.RedisConf
	// es config
	// Es struct {
	// 	Addresses []string
	// 	Username  string
	// 	Password  string
	// }
	// UserRPC zrpc.RpcClientConf
}
