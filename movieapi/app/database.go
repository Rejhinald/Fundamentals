package app

import (
    "context"
    "log"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var DynamoDBClient *dynamodb.Client

func InitDynamoDB() {
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-southeast-1"))
    if err != nil {
        log.Fatalf("Unable to load AWS config: %v", err)
    }
    DynamoDBClient = dynamodb.NewFromConfig(cfg)
}
