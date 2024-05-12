package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	l "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"os"
	"strings"
	"sync"
)

const failFunction = "Function failed"
const okFunction = "Function successful"

var ec2Client *ec2.Client
var snsClient *sns.Client
var lambdaClient *l.Client
var topicArn string

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("REGION")))
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	ec2Client = ec2.NewFromConfig(cfg)
	snsClient = sns.NewFromConfig(cfg)
	lambdaClient = l.NewFromConfig(cfg)
	topicArn = os.Getenv("TOPIC_ARN")
	if err != nil {
		panic("configuration error, " + err.Error())
	}
}

func tagCheck(tags []types.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == "CostCenter" {
			return true
		}
	}
	return false
}

func checkEBSInstances() ([]string, error) {
	var nextToken *string
	var resourcesMissingTags []string

	for {
		input := &ec2.DescribeVolumesInput{
			MaxResults: aws.Int32(100),
			NextToken:  nextToken,
		}
		result, err := ec2Client.DescribeVolumes(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, volume := range result.Volumes {
			if foundTag := tagCheck(volume.Tags); !foundTag {
				resourcesMissingTags = append(resourcesMissingTags, *volume.VolumeId)
			}
		}
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}
	fmt.Println(resourcesMissingTags)
	return resourcesMissingTags, nil
}

func checkEC2Instances() ([]string, error) {
	var nextToken *string
	var resourcesMissingTags []string

	for {
		input := &ec2.DescribeInstancesInput{
			Filters: []types.Filter{
				{
					Name: aws.String("instance-state-name"),
					Values: []string{
						"running",
						"stopped",
					},
				},
			},
			MaxResults: aws.Int32(100),
			NextToken:  nextToken,
		}
		result, err := ec2Client.DescribeInstances(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				if foundTag := tagCheck(instance.Tags); !foundTag {
					resourcesMissingTags = append(resourcesMissingTags, *instance.InstanceId)
				}
			}
		}
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}
	fmt.Println(resourcesMissingTags)
	return resourcesMissingTags, nil
}

func checkSnapshots() ([]string, error) {
	var nextToken *string
	var resourcesMissingTags []string

	for {
		input := &ec2.DescribeSnapshotsInput{
			MaxResults: aws.Int32(100),
			OwnerIds:   []string{"self"},
			NextToken:  nextToken,
		}
		result, err := ec2Client.DescribeSnapshots(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, snap := range result.Snapshots {
			if foundTag := tagCheck(snap.Tags); !foundTag {
				resourcesMissingTags = append(resourcesMissingTags, *snap.SnapshotId)
			}
		}
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}
	fmt.Println(resourcesMissingTags)
	return resourcesMissingTags, nil
}

func checkLambdaFunctions() ([]string, error) {
	var nextMarker *string
	var resourcesMissingTags []string

	for {
		input := &l.ListFunctionsInput{
			MaxItems: aws.Int32(100),
			Marker:   nextMarker,
		}
		result, err := lambdaClient.ListFunctions(context.TODO(), input)
		if err != nil {
			return nil, err
		}
		for _, allFunctions := range result.Functions {
			res, err := lambdaClient.ListTags(context.TODO(), &l.ListTagsInput{
				Resource: allFunctions.FunctionArn,
			})
			if err != nil {
				return nil, err
			}
			if _, exists := res.Tags["CostCenter"]; !exists {
				resourcesMissingTags = append(resourcesMissingTags, *allFunctions.FunctionName)
			}
		}
		if result.NextMarker == nil {
			break
		}
		nextMarker = result.NextMarker
	}
	fmt.Println(resourcesMissingTags)
	return resourcesMissingTags, nil
}

func checkResources(resourceType string, checkFunc func() ([]string, error)) string {
	msg, err := checkFunc()
	if err != nil {
		fmt.Println(err)
		return err.Error()
	}
	if len(msg) > 0 {
		return fmt.Sprintf("%s without CostCenter tag: %v", resourceType, msg)
	}
	return ""
}

func HandleRequest() ([]string, error) {
	results := make(chan string, 4)
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		results <- checkResources("Snapshots", checkSnapshots)
	}()

	go func() {
		defer wg.Done()
		results <- checkResources("Instances", checkEC2Instances)
	}()

	go func() {
		defer wg.Done()
		results <- checkResources("Volumes", checkEBSInstances)
	}()

	go func() {
		defer wg.Done()
		results <- checkResources("Lambda", checkLambdaFunctions)
	}()

	wg.Wait()
	close(results)

	var out strings.Builder
	for result := range results {
		if result != "" {
			out.WriteString(result + "\n\n")
		}
	}

	_, err := snsClient.Publish(context.TODO(), &sns.PublishInput{
		Message:  aws.String(out.String()),
		TopicArn: aws.String(topicArn),
		Subject:  aws.String("Un-Tagged Resources"),
	})
	if err != nil {
		fmt.Println(err)
		return []string{failFunction}, err
	}
	return []string{okFunction}, nil
}

func main() {
	lambda.Start(HandleRequest)
	//_, _ = HandleRequest()
}
