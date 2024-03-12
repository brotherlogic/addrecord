package main

import (
	"context"
	"log"
	"strings"
	"time"
)

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
					if strings.Contains(comment, ".") {
						setPrice(detail, strings.Replace(comment, ".", "", -1))
					} else {
						setId(detail, comment)
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
					if strings.HasPrefix(label, "Parents") {
						setOrigin(detail, label)
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
