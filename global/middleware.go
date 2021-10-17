package global

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/xiang/data"
	"github.com/yaoapp/xiang/share"
)

// FileServer 静态服务
var FileServer http.Handler = http.FileServer(data.AssetFS())

// Middlewares 服务中间件
var Middlewares = []gin.HandlerFunc{
	BindDomain,
	BinStatic,
}

// BindDomain 绑定许可域名
func BindDomain(c *gin.Context) {

	for _, allow := range share.AllowHosts {
		if strings.Contains(c.Request.Host, allow) {
			c.Next()
			return
		}
	}

	c.JSON(403, gin.H{
		"code":    403,
		"message": fmt.Sprintf("%s is not allowed", c.Request.Host),
	})
	c.Abort()
}

// BinStatic 静态文件服务
func BinStatic(c *gin.Context) {
	if len(c.Request.URL.Path) >= 5 && c.Request.URL.Path[0:5] == "/api/" {
		c.Next()
		return
	}

	// 静态文件请求
	FileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
