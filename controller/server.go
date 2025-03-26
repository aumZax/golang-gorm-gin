package controller

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var DB *gorm.DB

func StartServer(db *gorm.DB) {
	DB = db
	gin.SetMode(gin.ReleaseMode) // ถ้าคอมมเม้น บรรทัดนี้จะเป็นโหมด debug ถ้าไม่ จะเป็นRelease mode
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "API is NOew",
		})
	})
	LoginRequestController(router)
	CartController(router, db)
	router.Run()
}
