package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Inexpediency/todo-rest-api/pkg"
	"github.com/Inexpediency/todo-rest-api/pkg/handler"
	"github.com/Inexpediency/todo-rest-api/pkg/repository"
	"github.com/Inexpediency/todo-rest-api/pkg/service"
)

func main() {
	logrus.SetFormatter(new(logrus.JSONFormatter))

	if err := initConfig(); err != nil {
		log.Fatalf("error occured while initializing config: %s", err)
	}

	if err := godotenv.Load("./.env"); err != nil {
		logrus.Fatalf("error loading env variables: %s", err)
	}

	cache := repository.NewRedisCache(repository.RedisConfig{
		Address: viper.GetString("cache.address"),
		Password: viper.GetString("cache.password"),
		DB: viper.GetInt("cache.db"),
	})

	db, err := repository.NewPostgresDB(repository.PostgresConfig{
		Host:     viper.GetString("db.host"),
		Port:     viper.GetString("db.port"),
		Username: viper.GetString("db.username"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   viper.GetString("db.dbname"),
		SSLMode:  viper.GetString("db.sslmode"),
	})

	if err != nil {
		logrus.Fatalf("failed to initalize db: %s", err)
	}

	repos := repository.NewRepository(db, cache)
	services := service.NewService(repos)
	handlers := handler.NewHandler(services)

	srv := new(pkg.Server)

	go func() {
		if err := srv.Run(viper.GetString("port"), handlers.InitRoutes()); err != nil {
			logrus.Fatalf("error occured while running the server: %s ", err)
		}
	}()

	logrus.Printf("todo-service has started on port=%d", viper.GetString("port"))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logrus.Print("todo-service is shutting down...")

	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.Errorf("error occured on server shutting down: %s", err)
	}

	if err := cache.Close(); err != nil {
		logrus.Errorf("error occured on closing cache connection: %s", err)
	}

	if err := db.Close(); err != nil {
		logrus.Errorf("error occured on closing db connection: %s", err)
	}
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")

	return viper.ReadInConfig()
}
