package logic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"beyond/application/article/api/internal/code"
	"beyond/application/article/api/internal/svc"
	"beyond/application/article/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 << 20 // 10MB

type UploadCoverLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadCoverLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadCoverLogic {
	return &UploadCoverLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadCoverLogic) UploadCover(req *http.Request) (*types.UploadCoverResponse, error) {
	// 解析请求中的multipart表单，maxFileSize定义了允许的最大文件大小。
	_ = req.ParseMultipartForm(maxFileSize)
	// file上传的文件内容、handler包含文件的元数据信息
	file, handler, err := req.FormFile("cover")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// 获取OSS存储桶对象，用于后续的文件上传。
	bucket, err := l.svcCtx.OssClient.Bucket(l.svcCtx.Config.Oss.BucketName)
	if err != nil {
		logx.Errorf("get bucket failed, err: %v", err)
		return nil, code.GetBucketErr
	}
	//生成上传文件的唯一名称。
	objectKey := genFilename(handler.Filename)
	//将文件内容上传到OSS存储桶。
	err = bucket.PutObject(objectKey, file)
	if err != nil {
		logx.Errorf("put object failed, err: %v", err)
		return nil, code.PutBucketErr
	}
	//返回上传成功后的文件URL。
	return &types.UploadCoverResponse{CoverUrl: genFileURL(objectKey)}, nil
}

func genFilename(filename string) string {
	return fmt.Sprintf("%d_%s", time.Now().UnixMilli(), filename)
}

func genFileURL(objectKey string) string {
	return fmt.Sprintf("https://zero-demo-ckx.oss-cn-hangzhou.aliyuncs.com/%s", objectKey)
}
