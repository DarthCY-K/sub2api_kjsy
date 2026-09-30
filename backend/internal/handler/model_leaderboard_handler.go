package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ModelLeaderboardHandler 「模型排行榜」：当月与历史累计的全站模型调用量排行。
//
// 路由挂 JWT：仅登录用户可见；数据为全站聚合（不含任何用户/Key 明细）。
// 费用字段（标准计费/实际扣费）仅管理员返回。
type ModelLeaderboardHandler struct {
	svc *service.ModelLeaderboardService
}

func NewModelLeaderboardHandler(svc *service.ModelLeaderboardService) *ModelLeaderboardHandler {
	return &ModelLeaderboardHandler{svc: svc}
}

// Get GET /api/v1/model-leaderboard?source=requested|upstream&month=YYYY-MM&metric=requests|tokens
func (h *ModelLeaderboardHandler) Get(c *gin.Context) {
	source := strings.TrimSpace(c.Query("source"))
	switch source {
	case "", "requested", "upstream":
	default:
		response.BadRequest(c, "invalid source, want requested|upstream")
		return
	}
	month := strings.TrimSpace(c.Query("month"))
	metric, err := service.ParseLeaderboardMetric(strings.TrimSpace(c.Query("metric")))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	role, _ := middleware.GetUserRoleFromContext(c)
	isAdmin := role == service.RoleAdmin

	data, err := h.svc.Get(c.Request.Context(), source, month, metric, isAdmin)
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid month") {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, "Failed to load model leaderboard")
		return
	}
	response.Success(c, data)
}
