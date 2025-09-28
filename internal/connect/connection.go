package connect

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/supabase-community/supabase-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	SupabaseClient *supabase.Client
	MongoDBClient  *mongo.Client
	Cld            *cloudinary.Cloudinary
)

// supabase init
func InitSupabase() (*supabase.Client, string, string, error) {
	url := os.Getenv("SUPABASE_URL")
	key := os.Getenv("SUPABASE_URL_ANON_KEY")
	client, err := supabase.NewClient(url, key, nil)
	if err != nil {
		return nil, "", "", err
	}
	SupabaseClient = client
	return client, url, key, nil
}

func Disconnect() {
	SupabaseClient = nil
}

// mongo init

func MongoDBConnect() (*mongo.Client, error) {
	uri := os.Getenv("MONGODB_URI")
	password := os.Getenv("MONGODB_PASSWORD")
	fullUri := strings.Replace(uri, "<password>", password, 1)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	clientOptions := options.Client().ApplyURI(fullUri)

	var err error
	MongoDBClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	if err := MongoDBClient.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	fmt.Println("✅ MongoDB connected successfully")
	return MongoDBClient, nil
}

func MongoDBDisconnect() error {
	if MongoDBClient == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := MongoDBClient.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect MongoDB: %v", err)
	}
	MongoDBClient = nil
	return nil
}

func CloudinaryCredentials() (*cloudinary.Cloudinary, error) {
	cloudinaryName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	cld, err := cloudinary.NewFromParams(
		cloudinaryName,
		apiKey,
		apiSecret,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cloudinary: %v", err)
	}

	fmt.Println("✅ Cloudinary connected successfully")
	return cld, nil
}
