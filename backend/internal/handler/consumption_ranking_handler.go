package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ConsumptionRankingHandler 每日消费排行榜公开接口。
type ConsumptionRankingHandler struct {
	rankingService *service.ConsumptionRankingService
}

// NewConsumptionRankingHandler 创建 ConsumptionRankingHandler。
func NewConsumptionRankingHandler(rankingService *service.ConsumptionRankingService) *ConsumptionRankingHandler {
	return &ConsumptionRankingHandler{rankingService: rankingService}
}

// GetToday 返回今日消费 Top 20（公开，无需鉴权）。
// GET /api/v1/public/ranking/consumption
func (h *ConsumptionRankingHandler) GetToday(c *gin.Context) {
	ranking, err := h.rankingService.GetToday(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, ranking)
}
