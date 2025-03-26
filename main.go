package main

import (
	"fmt"
	"go2006/controller"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(viper.Get("mysql.dsn"))
	dsn := viper.GetString("mysql.dsn")

	dialactor := mysql.Open(dsn)
	db, err := gorm.Open(dialactor, &gorm.Config{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Connection successful")

	// customers := []model.Customer{}
	// result := db.Find(&customers)
	// if result.Error != nil {
	// 	panic(result.Error)
	// }
	// fmt.Printf("%v", customers)
	controller.StartServer(db)

}
