package controller

import (
	"go2006/model"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func LoginRequestController(router *gin.Engine) {
	routes := router.Group("/customer")
	{
		routes.GET("", ping)
		routes.POST("/login", login)                  // เส้นทางสำหรับล็อกอิน
		routes.GET("/profile/:id", getProfile)        // เส้นทางสำหรับดูโปรไฟล์
		routes.PUT("/profile/:id", updateAddress)     // เส้นทางสำหรับแก้ไขที่อยู่
		routes.PUT("/changepass/:id", changePassword) // เพิ่มเส้นทางสำหรับเปลี่ยนรหัสผ่าน

	}
}

// ฟังก์ชันตรวจสอบการล็อกอิน
func login(c *gin.Context) {
	var loginData struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var customer model.Customer
	if err := DB.Where("email = ?", loginData.Email).First(&customer).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(customer.Password), []byte(loginData.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// สร้าง response ตามรูปแบบในตัวอย่าง
	response := gin.H{
		"CustomerID":  customer.CustomerID,
		"FirstName":   customer.FirstName,
		"LastName":    customer.LastName,
		"Email":       customer.Email,
		"PhoneNumber": customer.PhoneNumber,
		"Address":     customer.Address,
		"CreatedAt":   customer.CreatedAt.Format(time.RFC3339),
		"UpdatedAt":   customer.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// ฟังก์ชันดึงข้อมูลโปรไฟล์
func getProfile(c *gin.Context) {
	id := c.Param("id")

	var customer model.Customer
	if err := DB.Where("customer_id = ?", id).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// ส่งกลับข้อมูลโดยไม่รวมรหัสผ่าน
	response := gin.H{
		"customer": gin.H{
			"customer_id":  customer.CustomerID,
			"first_name":   customer.FirstName,
			"last_name":    customer.LastName,
			"email":        customer.Email,
			"phone_number": customer.PhoneNumber,
			"address":      customer.Address,
		},
	}
	c.JSON(http.StatusOK, response)
}

// ฟังก์ชันแก้ไขที่อยู่
func updateAddress(c *gin.Context) {
	id := c.Param("id")

	var updateData struct {
		Address string `json:"address" binding:"required"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// อัปเดตที่อยู่ในฐานข้อมูล
	result := DB.Model(&model.Customer{}).Where("customer_id = ?", id).Update("address", updateData.Address)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address updated successfully",
		"address": updateData.Address,
	})
}

func ping(c *gin.Context) {
	var customers []model.Customer
	if err := DB.Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch customers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Successfully retrieved customers",
		"customers": customers,
	})
}

// ฟังก์ชันสำหรับเปลี่ยนรหัสผ่าน
func changePassword(c *gin.Context) {
	id := c.Param("id")

	var changePassData struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&changePassData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ดึงข้อมูลลูกค้าจากฐานข้อมูล
	var customer model.Customer
	if err := DB.Where("customer_id = ?", id).First(&customer).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// ตรวจสอบรหัสผ่านเก่า
	if err := bcrypt.CompareHashAndPassword([]byte(customer.Password), []byte(changePassData.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Old password is incorrect"})
		return
	}

	// เข้ารหัสรหัสผ่านใหม่
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(changePassData.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	// อัปเดตรหัสผ่านใหม่ในฐานข้อมูล
	if err := DB.Model(&customer).Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}
