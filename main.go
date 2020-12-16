package main

import (
	"context"
	"fmt"
	"os"

	registrator "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LambdaFunction struct {
	Function   string   `json:"function" bson:"function"`
	Region     string   `json:"region" bson:"region"`
	Qualifiers []string `json:"qualifiers" bson:"qualifiers"`
	Instances  int      `json:"instances" bson:"instances"`
}

var ctx context.Context
var err error
var client *mongo.Client

func init() {
	ctx = context.Background()
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
}

func handler() error {
	collection := client.Database(os.Getenv("MONGO_DATABASE")).Collection("functions")
	cur, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	functions := make([]LambdaFunction, 0)
	for cur.Next(ctx) {
		var function LambdaFunction
		cur.Decode(&function)
		functions = append(functions, function)
	}

	cfg, _ := config.LoadDefaultConfig()

	for _, function := range functions {
		cfg.Region = function.Region
		svc := lambda.NewFromConfig(cfg)

		for _, qualifier := range function.Qualifiers {
			fmt.Println(function.Function, " - ", qualifier)
			for i := 0; i < function.Instances; i++ {
				fmt.Println((i + 1), " request")
				svc.Invoke(context.Background(), &lambda.InvokeInput{
					FunctionName: aws.String(function.Function),
					Qualifier:    aws.String(qualifier),
				})
			}
		}
	}

	return nil
}

func main() {
	registrator.Start(handler)
}
