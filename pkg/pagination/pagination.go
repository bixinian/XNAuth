package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Params struct {
	Page     int
	PageSize int
	Offset   int
}

func FromQuery(c *gin.Context) Params {
	page := parsePositiveInt(c.Query("page"), DefaultPage)
	pageSize := parsePositiveInt(c.Query("page_size"), DefaultPageSize)
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return Params{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
