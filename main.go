package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	ghbclient "github.com/brotherlogic/githubridge/client"
	rsclient "github.com/brotherlogic/rstore/client"

	pb "github.com/brotherlogic/addrecord/proto"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	rspb "github.com/brotherlogic/rstore/proto"
)

func loadConfig(ctx context.Context) (*pb.Config, error) {
	rsclient, err := rsclient.GetClient()
	if err != nil {
		return nil, err
	}

	data, err := rsclient.Read(ctx, &rspb.ReadRequest{})
	if err != nil {
		return nil, err
	}

	config := &pb.Config{}
	err = proto.Unmarshal(data.GetValue().GetValue(), config)
	return config, err
}

func seen(config *pb.Config, photo string) (*pb.Tracker, error) {
	for _, tracker := range config.GetTrackers() {
		if tracker.PhotoId == photo {
			return tracker, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "Unable to locate %v", photo)
}

func needsCommentDetail(tracker *pb.Tracker) bool {
	return tracker.GetCost() == 0 || tracker.GetId() == 0
}

func needsLabelDetail(tracker *pb.Tracker) bool {
	return tracker.GetFolder() == ""
}

func getComments(ctx context.Context, tracker *pb.Tracker) ([]*ghbpb.Comment, error) {
	client, err := ghbclient.GetClientInternal()
	if err != nil {
		return nil, err
	}

	issue, err := client.GetIssue(ctx, &ghbpb.GetIssueRequest{
		Repo: "addrecord",
		User: "brotherlogic",
		Id:   int32(tracker.GetId()),
	})
	if err != nil {
		return nil, err
	}

	return comments.GetComments(), err
}

func getLabels(ctx context.Context, tracker *pb.Tracker) ([]string, error) {
	client, err := ghbclient.GetClientInternal()
	if err != nil {
		return nil, err
	}

	labels, err := client.GetLabels(ctx, &ghbpb.GetLabelsRequest{
		Repo: "addrecord",
		User: "brotherlogic",
		Id:   int32(tracker.GetId()),
	})
	if err != nil {
		return nil, err
	}

	return labels.GetLabels(), nil
}

func addIssue(ctx context.Context, config *pb.Config, photo string) error {
	client, err := ghbclient.GetClientInternal()
	if err != nil {
		return err
	}

	issue, err := client.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
		Repo:  "addrecord",
		User:  "brotherlogic",
		Title: "Record To Add",
		Body:  fmt.Sprintf("%v", photo),
	})
	if err != nil {
		return err
	}

	config.Trackers = append(config.Trackers, &pb.Tracker{
		IssueId: int32(issue.GetIssueId()),
		PhotoId: photo,
	})

	return err
}

func setPrice(tracker *pb.Tracker, price string) {
	val, _ := strconv.ParseInt(price, 10, 32)
	tracker.Cost = int32(val)
}

func setId(tracker *pb.Tracker, id string) {
	val, _ := strconv.ParseInt(id, 10, 64)
	tracker.Id = val
}

func setDestination(tracker *pb.Tracker, dest string) {
	tracker.Folder = dest
}

func setLocation(tracker *pb.Tracker, location string) {
	tracker.Location = location
}

func readyToAdd(tracker *pb.Tracker) bool {
	return tracker.GetCost() == 0 ||
		tracker.GetFolder() == "" ||
		tracker.GetId() == 0 ||
		tracker.GetLocation() == ""
}

func main() {
	log.Printf("DOing a record add run")
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	// Load the config
	config, err := loadConfig(ctx)
	if err != nil {
		log.Fatalf("Unable to get config: %v", err)
	}

	photos, err := getAlbum(ctx, "Records To Add")
	if err != nil {
		log.Fatalf("Unable to get albums: %v", err)
	}

	for _, photo := range photos {
		if detail, err := seen(config, photo); err == nil {
			// Handle comments
			if needsCommentDetail(detail) {
				comments, err := getComments(ctx, detail)
				if err != nil {
					log.Fatalf("Cannot get comments: %v", err)
				}
				for _, comment := range comments {
					if strings.Contains(comment.GetText(), ".") {
						setPrice(detail, strings.Replace(comment.GetText(), ".", "", -1))
					} else {
						setId(detail, comment.GetText())
					}
				}
			}

			// Handle tags
			if needsLabelDetail(detail) {
				labels, err := getLabels(ctx, detail)
				if err != nil {
					log.Fatalf("canntot get labels: %v", labels)
				}

				for _, label := range labels {
					if strings.HasPrefix(label, "Location: ") {
						setLocation(detail, label)
					}
					if strings.HasPrefix(label, "Destination: ") {
						setDestination(detail, label)
					}
				}
			}

			if readyToAdd(detail) {
				add(ctx, config, detail)
			}
		} else {
			addIssue(ctx, config, photo)
		}
	}
}
