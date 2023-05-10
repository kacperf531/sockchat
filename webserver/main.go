package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/joho/godotenv"
	"github.com/kacperf531/sockchat"
	"github.com/kacperf531/sockchat/services"
	"github.com/kacperf531/sockchat/storage"
	"github.com/redis/go-redis/v9"
)

const (
	defaultTimeoutUnauthorized = 1 * time.Minute
	defaultTimeoutAuthorized   = 10 * time.Minute
	grpcPort                   = 50051
)

func main() {
	godotenv.Load("../.env")
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)

	TestRedisConnection()
	TestElasticSearchConnection()

	mySqlDb := mustConnectToMySql()
	TestMySqlConnection(mySqlDb)
	resetMySqlForDevEnvironment(mySqlDb)
	es := mustInitializeElasticSearchClient()

	messageStore := storage.NewMessageStore(es, os.Getenv("ES_MESSAGES_INDEX"))
	channelStore := sockchat.NewChannelStore(messageStore)
	userStore := storage.NewUserStore(mySqlDb)

	userCache := mustInitializeRedisClient()
	userProfileService := &sockchat.ProfileService{Store: userStore, Cache: userCache}
	connectedUsers := sockchat.NewConnectedUsersPool(channelStore)

	authService := &services.SockchatAuthService{UserProfiles: userProfileService}
	coreService := &services.SockchatCoreService{
		UserProfiles:   userProfileService,
		Messages:       messageStore,
		ChatChannels:   channelStore,
		ConnectedUsers: connectedUsers}

	httpRouter := http.NewServeMux()

	webAPI := services.NewWebAPI(coreService, authService)
	webAPI.HandleRequests(httpRouter)
	grpcAPI := services.NewSockchatGRPCServer(coreService, authService)
	services.ServeGRPC(grpcAPI, grpcPort)
	messagingAPI := &services.MessagingAPI{TimeoutAuthorized: defaultTimeoutAuthorized, TimeoutUnauthorized: defaultTimeoutUnauthorized, ConnectedUsers: connectedUsers, UserProfiles: userProfileService}
	messagingAPI.HandleRequests(httpRouter)

	log.Fatal(http.ListenAndServe(":8080", httpRouter))
}

func mustConnectToMySql() *sql.DB {
	db, err := sql.Open("mysql", os.Getenv("DB_USER")+":"+os.Getenv("DB_PASSWORD")+"@tcp("+os.Getenv("DB_HOST")+")/sockchat")
	if err != nil {
		log.Fatalf("could not create db object due to an error: %v", err)
	}
	return db
}

func TestMySqlConnection(mysqlDB *sql.DB) {
	err := mysqlDB.Ping()
	if err != nil {
		log.Fatalf("could not connect to the db due to an error: %v", err)
	}
}

func resetMySqlForDevEnvironment(mysqlDB *sql.DB) {
	if os.Getenv("ENVIRONMENT") == "DEV" {
		err := storage.ResetUsersTable(mysqlDB)
		if err != nil {
			log.Fatalf("error setting up the users table %v", err)
		}
	}
}

func mustInitializeElasticSearchClient() *elasticsearch.Client {
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		log.Fatalf("Error creating the elasticsearch client: %s", err)
	}
	return es
}

func mustInitializeRedisClient() *redis.Client {
	dbIndex, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Fatalf("could not parse redis db index: %v", err)
	}
	return redis.NewClient(
		&redis.Options{
			Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       dbIndex,
		},
	)
}
