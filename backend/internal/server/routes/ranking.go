package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRankingRoutes 注册每日消费排行榜公开路由（无需鉴权）。
func RegisterRankingRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	ranking := v1.Group("/public/ranking")
	ranking.Use(panelRateLimiter.PublicIP())
	{
		ranking.GET("/consumption", h.ConsumptionRanking.GetToday)
	}
}
