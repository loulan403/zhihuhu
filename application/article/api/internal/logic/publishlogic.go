package logic

import (
	"context"
	"encoding/json"

	"beyond/application/article/api/internal/code"
	"beyond/application/article/api/internal/svc"
	"beyond/application/article/api/internal/types"
	"beyond/application/article/rpc/pb"
	"beyond/pkg/xcode"

	"github.com/zeromicro/go-zero/core/logx"
)

const minContentLen = 80

type PublishLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishLogic) Publish(req *types.PublishRequest) (*types.PublishResponse, error) {
	//内容限制
	if len(req.Title) == 0 {
		return nil, code.ArtitleTitleEmpty
	}
	if len(req.Content) < minContentLen {
		return nil, code.ArticleContentTooFewWords
	}
	if len(req.Cover) == 0 {
		return nil, code.ArticleCoverEmpty
	}
	// 获取用户id，因为userId是string，这里断言成int64
	userId, err := l.ctx.Value("userId").(json.Number).Int64()
	if err != nil {
		logx.Errorf("l.ctx.Value error: %v", err)
		return nil, xcode.NoLogin
	}
	//调用ArticleRPC服务的Publish方法发布文章
	pret, err := l.svcCtx.ArticleRPC.Publish(l.ctx, &pb.PublishRequest{
		UserId:      userId,
		Title:       req.Title,
		Content:     req.Content,
		Description: req.Description,
		Cover:       req.Cover,
	})
	if err != nil {
		logx.Errorf("l.svcCtx.ArticleRPC.Publish req: %v userId: %d error: %v", req, userId, err)
		return nil, err
	}
	// 返回发布成功后的文章ID
	return &types.PublishResponse{ArticleId: pret.ArticleId}, nil
}
