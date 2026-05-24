package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
)

func ParseListVideosQuery(c *gin.Context) dto.ListVideosQuery {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "0"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	return dto.ListVideosQuery{
		Search: c.Query("q"),
		Status: c.DefaultQuery("status", "all"),
		Filter: c.DefaultQuery("filter", "all"),
		Days:   days,
		Sort:   c.DefaultQuery("sort", "processed_at"),
		Order:  c.DefaultQuery("order", "desc"),
		Page:   page,
		Limit:  limit,
	}
}
