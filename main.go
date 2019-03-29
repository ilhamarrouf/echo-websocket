package main

import (
	"database/sql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pusher/pusher-http-go"
	"io"
	"net/http"
	"os"
)

/**
Pusher
 */
var client = pusher.Client{
	AppId: "747415",
	Key: "8f1f1df84f6454ee44c3",
	Secret: "7e478690d12810e3b550",
	Cluster: "ap1",
	Secure: true,
}

/**
Structs
 */
type Photo struct {
	ID int64 `json:"id"`
	Src string `json:"src"`
}

type PhotoCollection struct {
	Photos []Photo `json:"items"`
}

/**
Handlers
 */
func getPhotos(db *sql.DB) echo.HandlerFunc {
	return func(context echo.Context) error {
		rows, err := db.Query("SELECT * FROM photos ORDER BY id DESC ")
		if err != nil {
			panic(err)
		}

		defer rows.Close()

		result := PhotoCollection{}

		for rows.Next() {
			photo := Photo{}

			err2 := rows.Scan(&photo.ID, &photo.Src)
			if err2 != nil {
				panic(err2)
			}

			result.Photos = append(result.Photos, photo)
		}

		return context.JSON(http.StatusOK, result)
	}
}

func storePhoto(db *sql.DB) echo.HandlerFunc {
	return func(context echo.Context) error {
		file, err := context.FormFile("file")
		if err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		defer src.Close()

		filePath := "./public/uploads/"+file.Filename
		fileSrc := "http://127.0.0.1:8888/uploads/"+file.Filename

		dst, err := os.Create(filePath)
		if err != nil {
			panic(err)
		}

		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			panic(err)
		}

		stmt, err := db.Prepare("INSERT INTO photos (src) VALUES (?)")
		if err != nil {
			panic(err)
		}

		defer stmt.Close()

		result, err := stmt.Exec(fileSrc)
		if err != nil {
			panic(err)
		}

		insertedId, err := result.LastInsertId()
		if err != nil {
			panic(err)
		}

		photo := Photo{
			Src: fileSrc,
			ID: insertedId,
		}

		client.Trigger("photo-stream", "new-photo", photo)

		return context.JSON(http.StatusOK, photo)
	}
}

/**
Helpers
 */
func initializeDatabase(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil || db == nil {
		panic("Error connecting to database")
	}

	return db
}

func migrateDatabase(db *sql.DB)  {
	sql := `
		CREATE TABLE IF NOT EXISTS photos(
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			src VARCHAR NOT NULL
		);
	`

	_, err := db.Exec(sql)
	if err != nil {
		panic(err)
	}
}

/**
Main app
 */
func main()  {
	db := initializeDatabase("database/database.sqlite")
	migrateDatabase(db)

	app := echo.New()
	app.Use(middleware.Recover())
	app.Use(middleware.Logger())
	app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowMethods: []string{echo.GET, echo.POST, echo.PATCH, echo.PUT, echo.DELETE, echo.OPTIONS},
	}))

	app.File("/", "public/index.html")
	app.GET("/photos", getPhotos(db))
	app.POST("/photos", storePhoto(db))
	app.Static("/uploads", "public/uploads")

	app.Logger.Fatal(app.Start(":8888"))
}