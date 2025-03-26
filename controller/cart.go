package controller

import (
	"go2006/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CartController(router *gin.Engine, db *gorm.DB) {
	routes := router.Group("/cart")
	{
		routes.GET("/search", searchProducts(db))
		routes.POST("/:cartName/add", addToCart(db))
		routes.GET("/:cartName/items", getCartItems(db))
	}
}

// Search products by description and price range
func searchProducts(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get query parameters
		description := c.Query("description")
		minPrice := c.Query("minPrice")
		maxPrice := c.Query("maxPrice")

		// Build query
		query := db.Model(&model.Product{})

		if description != "" {
			query = query.Where("description LIKE ?", "%"+description+"%")
		}

		if minPrice != "" {
			if min, err := strconv.ParseFloat(minPrice, 64); err == nil {
				query = query.Where("price >= ?", min)
			}
		}

		if maxPrice != "" {
			if max, err := strconv.ParseFloat(maxPrice, 64); err == nil {
				query = query.Where("price <= ?", max)
			}
		}

		// Execute query
		var products []model.Product
		if err := query.Find(&products).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search products"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"products": products,
		})
	}
}

// Add product to cart
func addToCart(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get parameters from request
		cartName := c.Param("cartName")
		customerID := c.GetInt("customerID") // Assume customerID is set from authentication middleware

		var request struct {
			ProductID int `json:"product_id" binding:"required"`
			Quantity  int `json:"quantity" binding:"required,min=1"`
		}

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Start transaction
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// Find or create cart
		var cart model.Cart
		if err := tx.Where("customer_id = ? AND cart_name = ?", customerID, cartName).First(&cart).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new cart if not exists
				cart = model.Cart{
					CustomerID: customerID,
					CartName:   cartName,
				}
				if err := tx.Create(&cart).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cart"})
					return
				}
			} else {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find cart"})
				return
			}
		}

		// Check if product exists
		var product model.Product
		if err := tx.First(&product, request.ProductID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		// Check stock availability
		if product.StockQuantity < request.Quantity {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
			return
		}

		// Find existing cart item
		var cartItem model.CartItem
		if err := tx.Where("cart_id = ? AND product_id = ?", cart.CartID, request.ProductID).First(&cartItem).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Create new cart item if not exists
				cartItem = model.CartItem{
					CartID:    cart.CartID,
					ProductID: request.ProductID,
					Quantity:  request.Quantity,
				}
				if err := tx.Create(&cartItem).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to cart"})
					return
				}
			} else {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check cart items"})
				return
			}
		} else {
			// Update quantity if item already exists
			cartItem.Quantity += request.Quantity
			if err := tx.Save(&cartItem).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart item"})
				return
			}
		}

		// Update product stock
		product.StockQuantity -= request.Quantity
		if err := tx.Save(&product).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product stock"})
			return
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Product added to cart successfully",
			"cart_id": cart.CartID,
		})
	}
}

// Get cart items
func getCartItems(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cartName := c.Param("cartName")
		customerID := c.GetInt("customerID") // Assume customerID is set from authentication middleware

		// Find cart
		var cart model.Cart
		if err := db.Where("customer_id = ? AND cart_name = ?", customerID, cartName).First(&cart).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Cart not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find cart"})
			return
		}

		// Get cart items with product details
		var cartItems []struct {
			model.CartItem
			ProductName string `json:"product_name"`
			Price       string `json:"price"`
		}

		if err := db.Table("cart_items").
			Select("cart_items.*, products.product_name, products.price").
			Joins("JOIN products ON products.product_id = cart_items.product_id").
			Where("cart_items.cart_id = ?", cart.CartID).
			Scan(&cartItems).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart items"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"cart_name": cart.CartName,
			"items":     cartItems,
		})
	}
}
