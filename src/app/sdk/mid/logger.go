package mid

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		now := time.Now()

		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = fmt.Sprintf("%s?%s", path, c.Request.URL.RawQuery)
		}

		log.Info(c, "request started", "method", c.Request.Method, "path", path, "remoteaddr", c.Request.RemoteAddr)

		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		log.Info(c, "request completed", "method", c.Request.Method, "path", path, "remoteaddr", c.Request.RemoteAddr,
			"statuscode", c.Writer.Status(), "since", time.Since(now).String())
	}
}
