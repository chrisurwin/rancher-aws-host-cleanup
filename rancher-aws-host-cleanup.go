package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/rancher/go-rancher/v2"
)

func main() {
	logrus.Info("Starting Rancher AWS Host cleanup")
	go startHealthcheck()
	for {
		go forever()
		time.Sleep(60 * time.Second)
	}
}

func forever() {
	logrus.Info("Checking for hosts to remove")
	//Get Rancher Variables
	cattleURL := os.Getenv("CATTLE_URL")
	if len(cattleURL) == 0 {
		logrus.Fatalf("CATTLE_URL is not set")
	}

	cattleAccessKey := os.Getenv("CATTLE_ACCESS_KEY")
	if len(cattleAccessKey) == 0 {
		logrus.Fatalf("CATTLE_ACCESS_KEY is not set")
	}

	cattleSecretKey := os.Getenv("CATTLE_SECRET_KEY")
	if len(cattleSecretKey) == 0 {
		logrus.Fatalf("CATTLE_SECRET_KEY is not set")
	}

	opts := &client.ClientOpts{
		Url:       cattleURL,
		AccessKey: cattleAccessKey,
		SecretKey: cattleSecretKey,
	}

	c, err := client.NewRancherClient(opts)

	if err != nil {
		logrus.Error("Error with client connection")
	}

	//Get a list of Hosts
	hosts, err := c.Host.List(nil)
	if err != nil {
		logrus.Error("Error with host list")
	}

	//Check if Host is in disconnected or reconnecting state
	for _, h := range hosts.Data {
		if h.State == "disconnected" || h.State == "reconnecting" || h.State == "inactive" {
			awsaz := strings.Split(h.Hostname, ".")[1]
			if validRegion(awsaz) {
				if hostTerminated(h.Hostname, awsaz) {
					//Need to set the host as inactive
					if h.State == "disconnected" || h.State == "reconnecting" {
						_, err := c.Host.ActionDeactivate(&h)
						if err != nil {
							logrus.Error("Error deactivating host: ", h.Hostname)
						} else {
							logrus.Info("Successfully deactivated host: ", h.Hostname)
						}
					}
					//If inactive then Remove
					if h.State == "inactive" {
						_, err := c.Host.ActionRemove(&h)
						if err != nil {
							logrus.Error("Error removing host: ", h.Hostname)
						} else {
							logrus.Info("Successfully removed host: ", h.Hostname)
						}
					}
				} else {
					logrus.Info("Host still valid: ", h.Hostname)
				}
			} else {
				logrus.Info("Either invalid region or not AWS host: ", h.Hostname)
			}

		}
	}
	logrus.Info("Waiting...")
}

//Function to check validity of aws region
func validRegion(r string) bool {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	svc := ec2.New(sess, &aws.Config{Region: aws.String(r)})
	if err != nil {
		panic(err)
	}
	regions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})

	if err != nil {
		logrus.Info("Invalid Region specified:", r)
	}
	for _, region := range regions.Regions {
		//Check that a valid region has been passed
		if *region.RegionName == r {
			return true
		}
	}
	return false
}

//Function to check if host has been terminated returns true if no longer in AWS or is in terminated state
func hostTerminated(h string, r string) bool {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	svc := ec2.New(sess, &aws.Config{Region: aws.String(r)})
	if err != nil {
		panic(err)
	}
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{Filters: []*ec2.Filter{&ec2.Filter{Name: aws.String("private-dns-name"), Values: []*string{aws.String(h)}}}})
	if err != nil {
		panic(err)
	}
	fmt.Println(len(resp.Reservations))
	if len(resp.Reservations) == 0 {
		return true
	} else if len(resp.Reservations) == 1 {
		if *resp.Reservations[0].Instances[0].State.Name == "terminated" {
			return true
		}
	} else {
		return false
	}
	return false
}
