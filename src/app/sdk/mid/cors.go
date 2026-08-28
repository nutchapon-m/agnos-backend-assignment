package mid

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Cors(origins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
