package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/response"
	middleware2 "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
)

type SubscriptionQuotaResetMonitorHandler struct {
	service *service.SubscriptionQuotaResetMonitorService
}

func NewSubscriptionQuotaResetMonitorHandler(svc *service.SubscriptionQuotaResetMonitorService) *SubscriptionQuotaResetMonitorHandler {
	return &SubscriptionQuotaResetMonitorHandler{service: svc}
}

type subscriptionQuotaResetMonitorRequest struct {
	Name                 string  `json:"name" binding:"required,max=100"`
	Enabled              *bool   `json:"enabled"`
	ExecutionEnabled     *bool   `json:"execution_enabled"`
	IntervalSeconds      int     `json:"interval_seconds" binding:"required,min=60,max=3600"`
	DropThresholdPercent float64 `json:"drop_threshold_percent" binding:"required,min=1,max=100"`
	CreditPolicy         string  `json:"credit_policy" binding:"required,oneof=ignore propagate"`
	ResetDaily           bool    `json:"reset_daily"`
	ResetWeekly          bool    `json:"reset_weekly"`
	ResetMonthly         bool    `json:"reset_monthly"`
	ResetFiveHour        bool    `json:"reset_five_hour"`
	AccountIDs           []int64 `json:"account_ids" binding:"required,min=1"`
	SubscriptionIDs      []int64 `json:"subscription_ids" binding:"required,min=1"`
}

func monitorID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, "invalid monitor id")
		return 0, false
	}
	return id, true
}
func (h *SubscriptionQuotaResetMonitorHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}
func (h *SubscriptionQuotaResetMonitorHandler) Get(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
func requestToMonitor(req subscriptionQuotaResetMonitorRequest) *service.SubscriptionQuotaResetMonitor {
	enabled := false
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	execution := false
	if req.ExecutionEnabled != nil {
		execution = *req.ExecutionEnabled
	}
	return &service.SubscriptionQuotaResetMonitor{Name: req.Name, Enabled: enabled, ExecutionEnabled: execution, IntervalSeconds: req.IntervalSeconds, DropThresholdPercent: req.DropThresholdPercent, CreditPolicy: req.CreditPolicy, ResetDaily: req.ResetDaily, ResetWeekly: req.ResetWeekly, ResetMonthly: req.ResetMonthly, ResetFiveHour: req.ResetFiveHour, AccountIDs: req.AccountIDs, SubscriptionIDs: req.SubscriptionIDs}
}
func (h *SubscriptionQuotaResetMonitorHandler) Create(c *gin.Context) {
	var req subscriptionQuotaResetMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item := requestToMonitor(req)
	var actorID *int64
	if subject, ok := middleware2.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		actorID = &subject.UserID
	}
	if err := h.service.Create(c.Request.Context(), item, actorID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
func (h *SubscriptionQuotaResetMonitorHandler) Update(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	var req subscriptionQuotaResetMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	item := requestToMonitor(req)
	item.ID = id
	if err := h.service.Update(c.Request.Context(), item); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
func (h *SubscriptionQuotaResetMonitorHandler) Check(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	if err := h.service.Check(c.Request.Context(), id, true); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
func (h *SubscriptionQuotaResetMonitorHandler) Events(c *gin.Context) {
	id, ok := monitorID(c)
	if !ok {
		return
	}
	events, err := h.service.Events(c.Request.Context(), id, 50)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, events)
}
