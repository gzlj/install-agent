package handle

import (
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/common"
)

func ShowConfig(c *gin.Context) {
	c.JSON(200, common.BuildResponse(200, "", common.G_config))
	return
}
