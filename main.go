package main

import (
	_ "KemonoSearch/docs"
	"fmt"
	"os"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

type Creator struct {
	Link      string `json:"link"`
	CreatorID string `json:"id" gorm:"primaryKey"`
	Name      string `json:"name" gorm:"index"`
	Service   string `json:"service" gorm:"primaryKey"`
	Indexed   int    `json:"indexed"`
	Updated   int    `json:"updated"`
	Favorited int    `json:"favorited"`
}

type SearchResult struct {
	SearchQuery string
	Creators    []Creator
}

var CreatorsFileUrl = "https://kemono.su/api/v1/creators.txt"
var db *gorm.DB

func main() {
	var err error
	db, err = gorm.Open(gormlite.Open(GetDefaultEnv("KEMONOSEARCH_DB", "creators.db")))
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Creator{}); err != nil {
		panic(err)
	}

	go func() {
		for {
			fmt.Println("syncing creators...")
			if err := SyncCreators(); err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Hour * 24)
		}
	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.LoadHTMLGlob("web/templates/*")

	router.GET("/", func(ctx *gin.Context) {
		name := ctx.Query("name")
		var creators []Creator
		var err error
		if name != "" {
			err = db.Where("name LIKE ?", "%"+name+"%").Find(&creators).Error
		}
		if err != nil {
			ctx.HTML(500, "error.html", gin.H{
				"error": err.Error(),
			})
			return
		}
		ctx.HTML(200, "index.html", SearchResult{
			SearchQuery: name,
			Creators:    creators,
		})
	})
	router.GET("/api/creator", SearchCreators)
	router.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	router.Run(GetDefaultEnv("KEMONOSEARCH_ADDR", ":39808"))
}

// @Summary Search creators
// @Description Search creators by name
// @Accept json
// @Produce json
// @Param name query string true "Creator name"
// @Success 200 {array} Creator
// @Router /creator [get]
func SearchCreators(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		ctx.JSON(400, gin.H{"error": "name is required"})
		return
	}
	var creators []Creator
	if err := db.Where("name LIKE ?", "%"+name+"%").Find(&creators).Error; err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, creators)
}

func SyncCreators() error {
	resp, err := req.R().Get(CreatorsFileUrl)
	if err != nil {
		return err
	}
	if resp.IsErrorState() {
		return fmt.Errorf("http status: %s", resp.GetStatus())
	}
	defer resp.Body.Close()
	decoder := sonic.ConfigDefault.NewDecoder(resp.Body)
	creators := make([]Creator, 0)
	err = decoder.Decode(&creators)
	if err != nil {
		return err
	}

	fmt.Printf("decoded creators: %d\n", len(creators))

	var count int64
	if err := db.Model(&Creator{}).Count(&count).Error; err != nil {
		return err
	}
	fmt.Printf("total creators in db: %d\n", count)

	tx := db.Begin()
	for _, creator := range creators {
		switch creator.Service {
		case "patreon":
			creator.Link = fmt.Sprintf("https://kemono.su/patreon/user/%s", creator.CreatorID)
		case "fanbox":
			creator.Link = fmt.Sprintf("https://kemono.su/fanbox/user/%s", creator.CreatorID)
		case "discord":
			creator.Link = fmt.Sprintf("https://kemono.su/discord/server/%s", creator.CreatorID)
		case "fantia":
			creator.Link = fmt.Sprintf("https://kemono.su/fantia/user/%s", creator.CreatorID)
		case "boosty":
			creator.Link = fmt.Sprintf("https://kemono.su/boosty/user/%s", creator.CreatorID)
		case "subscribestar":
			creator.Link = fmt.Sprintf("https://kemono.su/subscribestar/user/%s", creator.CreatorID)
		case "dlsite":
			creator.Link = fmt.Sprintf("https://kemono.su/dlsite/user/%s", creator.CreatorID)
		}
		if err := tx.Save(&creator).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	if err := db.Model(&Creator{}).Count(&count).Error; err != nil {
		return err
	}
	fmt.Printf("update success, total creators in db: %d\n", count)
	return nil
}

func GetDefaultEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
