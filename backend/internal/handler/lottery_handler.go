package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// LotteryHandler 每日盲盒抽奖用户侧接口。
type LotteryHandler struct {
	lotteryService *service.LotteryService
}

// NewLotteryHandler 创建 LotteryHandler。
func NewLotteryHandler(lotteryService *service.LotteryService) *LotteryHandler {
	return &LotteryHandler{lotteryService: lotteryService}
}

// GetStatus 获取抽奖状态（可用次数、今日充值进度、是否已领登录奖励、当前余额）。
// GET /api/v1/lottery/status
func (h *LotteryHandler) GetStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	status, err := h.lotteryService.Status(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

// Draw 执行一次抽奖。
// POST /api/v1/lottery/draw
func (h *LotteryHandler) Draw(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.lotteryService.Draw(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// GetRecords 分页返回用户中奖记录。
// GET /api/v1/lottery/records
func (h *LotteryHandler) GetRecords(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	records, result, err := h.lotteryService.Records(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.PaginatedWithResult(c, records, &response.PaginationResult{
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
		Pages:    result.Pages,
	})
}

// GetAdminDailyStats 管理员查看每日抽奖统计（按上海自然日）。
// GET /api/v1/admin/lottery/stats
// 可选 query：start_date / end_date（YYYY-MM-DD，Asia/Shanghai），默认最近 30 天。
func (h *LotteryHandler) GetAdminDailyStats(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.lotteryService.AdminDailyStats(c.Request.Context(), startDate, endDate)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}
