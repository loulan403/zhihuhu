package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"beyond/application/article/rpc/internal/code"
	"beyond/application/article/rpc/internal/model"
	"beyond/application/article/rpc/internal/svc"
	"beyond/application/article/rpc/internal/types"
	"beyond/application/article/rpc/pb"

	"github.com/zeromicro/go-zero/core/mr"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	prefixArticles = "biz#articles#%d#%d"
	articlesExpire = 3600 * 24 * 2
)

type ArticlesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewArticlesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ArticlesLogic {
	return &ArticlesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ArticlesLogic) Articles(in *pb.ArticlesRequest) (*pb.ArticlesResponse, error) {
	fmt.Println(in.UserId, in.SortType)
	fmt.Println("已经读到传入查询数据")
	if in.SortType != types.SortPublishTime && in.SortType != types.SortLikeCount {
		return nil, code.SortTypeInvalid
	}
	if in.UserId <= 0 {
		return nil, code.UserIdInvalid
	}
	if in.PageSize == 0 {
		in.PageSize = types.DefaultPageSize
	}
	if in.Cursor == 0 {
		if in.SortType == types.SortPublishTime {
			in.Cursor = time.Now().Unix()
		} else {
			in.Cursor = types.DefaultSortLikeCursor
		}
	}

	var (
		sortField       string
		sortLikeNum     int64
		sortPublishTime string
	)
	if in.SortType == types.SortLikeCount {
		sortField = "like_num"
		sortLikeNum = in.Cursor
	} else {
		sortField = "publish_time"
		sortPublishTime = time.Unix(in.Cursor, 0).Format("2006-01-02 15:04:05")
	}

	var (
		err            error
		isCache, isEnd bool
		lastId, cursor int64
		curPage        []*pb.ArticleItem
		articles       []*model.Article
	)
	articleIds, _ := l.cacheArticles(l.ctx, in.UserId, in.Cursor, in.PageSize, in.SortType)
	//大于0说明命中了有效的缓存
	fmt.Println(articleIds, "为空则未中缓存")
	if len(articleIds) > 0 {
		isCache = true
		//如果最后一页为-1，已经到头了，设置isend
		if articleIds[len(articleIds)-1] == -1 {
			isEnd = true
		}
		//拿到文章id，根据id查找文章详情
		articles, err = l.articleByIds(l.ctx, articleIds)
		if err != nil {
			return nil, err
		}

		// // 通过sortFiled对articles进行排序
		// var cmpFunc func(a, b *model.Article) int
		// if sortField == "like_num" {
		// 	cmpFunc = func(a, b *model.Article) int {
		// 		return cmp.Compare(b.LikeNum, a.LikeNum)
		// 	}
		// } else {
		// 	cmpFunc = func(a, b *model.Article) int {
		// 		return cmp.Compare(b.PublishTime.Unix(), a.PublishTime.Unix())
		// 	}
		// }
		// slices.SortFunc(articles, cmpFunc)
		for _, article := range articles {
			curPage = append(curPage, &pb.ArticleItem{
				Id:           article.Id,
				Title:        article.Title,
				Content:      article.Content,
				LikeCount:    article.LikeNum,
				CommentCount: article.CommentNum,
				PublishTime:  article.PublishTime.Unix(),
			})
		}
		//没有命中，此时需要从数据库中查找
	} else {
		// 使用singleflight防止数据库击穿
		v, err, _ := l.svcCtx.SingleFlightGroup.Do(fmt.Sprintf("ArticlesByUserId:%d:%d", in.UserId, in.SortType), func() (interface{}, error) {
			return l.svcCtx.ArticleModel.ArticlesByUserId(l.ctx, in.UserId, types.ArticleStatusVisible, sortLikeNum, sortPublishTime, sortField, types.DefaultLimit)
		})
		// fmt.Println("查询参数：", in.UserId, sortLikeNum, sortPublishTime, sortField, types.DefaultLimit)
		//ArticlesByUserId(ctx context.Context, userId int64, status int, likeNum int64, pubTime, sortField string, limit int)
		// articles, err = l.svcCtx.ArticleModel.ArticlesByUserId(l.ctx, in.UserId, types.ArticleStatusVisible, sortLikeNum, sortPublishTime, sortField, types.DefaultLimit)
		// fmt.Println("1查询到信息articles：", articles)
		if err != nil {
			logx.Errorf("ArticlesByUserId userId: %d sortField: %s error: %v", in.UserId, sortField, err)
			return nil, err
		}
		if v == nil {
			return &pb.ArticlesResponse{}, nil
		}
		articles = v.([]*model.Article)
		var firstPageArticles []*model.Article
		//如果查询的数量大于页大小，则需要分页。否则是最后一页
		fmt.Println("2查询到信息articles：", articles)
		if len(articles) > int(in.PageSize) {
			firstPageArticles = articles[:int(in.PageSize)]
		} else {
			firstPageArticles = articles
			isEnd = true
		}
		for _, article := range firstPageArticles {
			curPage = append(curPage, &pb.ArticleItem{
				Id:           article.Id,
				Title:        article.Title,
				Content:      article.Content,
				LikeCount:    article.LikeNum,
				CommentCount: article.CommentNum,
				PublishTime:  article.PublishTime.Unix(),
			})
		}
		fmt.Println("3查询到信息articles：", articles)
	}
	fmt.Println("4查询到信息articles：", articles)
	// 如果当前页面有内容
	if len(curPage) > 0 {
		//获取当前页面的最后一篇文章
		pageLast := curPage[len(curPage)-1]
		lastId = pageLast.Id
		if in.SortType == types.SortPublishTime {
			cursor = pageLast.PublishTime
		} else {
			cursor = pageLast.LikeCount
		}
		if cursor < 0 {
			cursor = 0
		}
		//遍历当前页面的文章，寻找游标位置，进行截断
		for k, article := range curPage {
			if in.SortType == types.SortPublishTime {
				if article.PublishTime == in.Cursor && article.Id == in.ArticleId {
					curPage = curPage[k:]
					break
				}
			} else {
				if article.LikeCount == in.Cursor && article.Id == in.ArticleId {
					curPage = curPage[k:]
					break
				}
			}
		}
	}
	//构建返回的响应对象
	ret := &pb.ArticlesResponse{
		IsEnd:     isEnd,
		Cursor:    cursor,
		ArticleId: lastId,
		Articles:  curPage,
	}
	fmt.Println("是否需要写缓存:", !isCache)
	fmt.Println("5查询到信息articles：", articles)
	//如果不是从缓存获取数据，则更新缓存
	if !isCache {
		// 在后台线程中安全地执行缓存更新操作
		threading.GoSafe(func() {
			//如果文章数量小于200，则说明已经查询到头，在articles中添加-1作为结束标志
			if len(articles) < types.DefaultLimit && len(articles) > 0 {
				articles = append(articles, &model.Article{Id: -1})
			}
			fmt.Println("添加结束标记：", articles)
			//更新缓存中的文章列表
			err = l.addCacheArticles(context.Background(), articles, in.UserId, in.SortType)
			if err != nil {
				logx.Errorf("addCacheArticles error: %v", err)
			}
		})
	}

	return ret, nil
}

func (l *ArticlesLogic) articleByIds(ctx context.Context, articleIds []int64) ([]*model.Article, error) {
	// 使用MapReduce框架并行处理文章查询。generate\mapper\reducer
	//中括号[]用来指定泛型参数类型的,输入int64,中间类型是*model.Article,输出类型是[]*model.Article
	articles, err := mr.MapReduce[int64, *model.Article, []*model.Article](func(source chan<- int64) {
		for _, aid := range articleIds {
			if aid == -1 {
				continue
			}
			source <- aid
		}
	}, func(id int64, writer mr.Writer[*model.Article], cancel func(error)) {
		p, err := l.svcCtx.ArticleModel.FindOne(ctx, id)
		if err != nil {
			cancel(err)
			return
		}
		writer.Write(p)
	}, func(pipe <-chan *model.Article, writer mr.Writer[[]*model.Article], cancel func(error)) {
		var articles []*model.Article
		for article := range pipe {
			articles = append(articles, article)
		}
		writer.Write(articles)
	})
	if err != nil {
		return nil, err
	}

	return articles, nil
}

func articlesKey(uid int64, sortType int32) string {
	return fmt.Sprintf(prefixArticles, uid, sortType)
}

func (l *ArticlesLogic) cacheArticles(ctx context.Context, uid, cursor, ps int64, sortType int32) ([]int64, error) {
	key := articlesKey(uid, sortType)
	//为了避免缓存击穿，首先判断缓存存不存在
	b, err := l.svcCtx.BizRedis.ExistsCtx(ctx, key)
	if err != nil {
		logx.Errorf("ExistsCtx key: %s error: %v", key, err)
	}
	if b {
		//存在则进行一次续期
		err = l.svcCtx.BizRedis.ExpireCtx(ctx, key, articlesExpire)
		if err != nil {
			logx.Errorf("ExpireCtx key: %s error: %v", key, err)
		}
	}
	//从缓存中获取文章列表
	pairs, err := l.svcCtx.BizRedis.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, key, 0, cursor, 0, int(ps))
	if err != nil {
		logx.Errorf("ZrevrangebyscoreWithScoresAndLimit key: %s error: %v", key, err)
		return nil, err
	}
	//解析文章ID列表
	var ids []int64
	for _, pair := range pairs {
		id, err := strconv.ParseInt(pair.Key, 10, 64)
		if err != nil {
			logx.Errorf("strconv.ParseInt key: %s error: %v", pair.Key, err)
			return nil, err
		}
		ids = append(ids, id)
	}
	//返回文章ID列表
	return ids, nil
}

func (l *ArticlesLogic) addCacheArticles(ctx context.Context, articles []*model.Article, userId int64, sortType int32) error {
	if len(articles) == 0 {
		return nil
	}
	// 根据用户ID和排序类型生成缓存键。
	key := articlesKey(userId, sortType)
	// 遍历文章列表，为每个文章生成分数和键值对，并将其添加到缓存中。
	for _, article := range articles {
		var score int64
		if sortType == types.SortLikeCount {
			score = article.LikeNum
		} else if sortType == types.SortPublishTime && article.Id != -1 {
			score = article.PublishTime.Local().Unix()
		}
		if score < 0 {
			score = 0
		}
		// 使用 Zadd 命令将键值对添加到有序集合中。
		_, err := l.svcCtx.BizRedis.ZaddCtx(ctx, key, score, strconv.Itoa(int(article.Id)))
		if err != nil {
			return err
		}
	}
	//设置缓存项的过期时间。
	return l.svcCtx.BizRedis.ExpireCtx(ctx, key, articlesExpire)
}
