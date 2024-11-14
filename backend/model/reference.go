package model

import (
	"time"

	pb "bigkinds.or.kr/proto/event"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Reference struct {
	ID         string              `json:"id"`
	Attributes ReferenceAttributes `json:"attributes"`
}

type ReferenceAttributes struct {
	NewsID      string    `json:"news_id"`
	Title       string    `json:"title"`
	PublishedAt time.Time `json:"published_at"`
	Provider    string    `json:"provider"`
	Byline      string    `json:"byline"`
	Content     string    `json:"content,omitempty"`
}

func FromModelReferenceToProtoReference(reference Reference) *pb.Reference {
	return &pb.Reference{
		Id: reference.ID,
		Attributes: &pb.ReferenceAttributes{
			NewsId:      reference.Attributes.NewsID,
			Title:       reference.Attributes.Title,
			PublishedAt: timestamppb.New(reference.Attributes.PublishedAt),
			Provider:    reference.Attributes.Provider,
			Byline:      reference.Attributes.Byline,
			Content:     reference.Attributes.Content,
		},
	}
}

func FromModelReferencesToProtoReferences(references []Reference) []*pb.Reference {
	var protoReferences []*pb.Reference

	for _, reference := range references {
		protoReferences = append(protoReferences, FromModelReferenceToProtoReference(reference))
	}

	return protoReferences
}
