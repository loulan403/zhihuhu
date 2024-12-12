package logic

import (
	"context"
	"strconv"
	"time"

	"beyond/application/article/rpc/internal/code"
	"beyond/application/article/rpc/internal/model"
	"beyond/application/article/rpc/internal/svc"
	"beyond/application/article/rpc/internal/types"
	"beyond/application/article/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// func (l *PublishLogic) Publish(in *pb.PublishRequest) (*pb.PublishResponse, error) {
// 	// 调用model层插入文章信息，ret是一个 sql.Result 类型的对象
// 	ret, err := l.svcCtx.ArticleModel.Insert(l.ctx, &model.Article{
// 		AuthorId:    in.UserId,
// 		Title:       in.Title,
// 		Content:     in.Content,
// 		Description: in.Description,
// 		Cover:       in.Cover,
// 		PublishTime: time.Now(),
// 		CreateTime:  time.Now(),
// 		UpdateTime:  time.Now(),
// 	})
// 	if err != nil {
// 		l.Logger.Errorf("Publish Insert req: %v error: %v", in, err)
// 		return nil, err
// 	}
// 	//从 ret 对象中获取刚刚插入记录的自增ID。
// 	articleId, err := ret.LastInsertId()
// 	if err != nil {
// 		l.Logger.Errorf("LastInsertId error: %v", err)
// 		return nil, err
// 	}
// 	//返回发布成功后的文章ID
// 	return &pb.PublishResponse{ArticleId: articleId}, nil
// }

func (l *PublishLogic) Publish(in *pb.PublishRequest) (*pb.PublishResponse, error) {
	//参数校验
	if in.UserId <= 0 {
		return nil, code.UserIdInvalid
	}
	if len(in.Title) == 0 {
		return nil, code.ArticleTitleCantEmpty
	}
	if len(in.Content) == 0 {
		return nil, code.ArticleContentCantEmpty
	}
	// 插入文章
	ret, err := l.svcCtx.ArticleModel.Insert(l.ctx, &model.Article{
		AuthorId:    in.UserId,
		Title:       in.Title,
		Content:     in.Content,
		Description: in.Description,
		Cover:       in.Cover,
		Status:      types.ArticleStatusVisible, // 正常逻辑应该先审核状态
		PublishTime: time.Now(),
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	})
	if err != nil {
		l.Logger.Errorf("Publish Insert req: %v error: %v", in, err)
		return nil, err
	}

	articleId, err := ret.LastInsertId()
	if err != nil {
		l.Logger.Errorf("LastInsertId error: %v", err)
		return nil, err
	}

	var (
		articleIdStr   = strconv.FormatInt(articleId, 10)
		publishTimeKey = articlesKey(in.UserId, types.SortPublishTime)
		likeNumKey     = articlesKey(in.UserId, types.SortLikeCount)
	)
	//同步写缓存，对发布时间和点赞数分别进行写缓存操作
	b, _ := l.svcCtx.BizRedis.ExistsCtx(l.ctx, publishTimeKey)
	//如果b存在于缓存中
	if b {
		_, err = l.svcCtx.BizRedis.ZaddCtx(l.ctx, publishTimeKey, time.Now().Unix(), articleIdStr)
		if err != nil {
			logx.Errorf("ZaddCtx req: %v error: %v", in, err)
		}
	}
	b, _ = l.svcCtx.BizRedis.ExistsCtx(l.ctx, likeNumKey)
	if b {
		_, err = l.svcCtx.BizRedis.ZaddCtx(l.ctx, likeNumKey, 0, articleIdStr)
		if err != nil {
			logx.Errorf("ZaddCtx req: %v error: %v", in, err)
		}
	}

	return &pb.PublishResponse{ArticleId: articleId}, nil
}
