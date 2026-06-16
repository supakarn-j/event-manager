package service

import (
	"context"
	"event-manager/service/client"
	"event-manager/utils"
	"fmt"
	"log"
)

type APIService interface {
	CreateNewStream(ctx context.Context, streamName string) error
	ListStreams(ctx context.Context) ([]client.StreamListItem, string, error)
	GetStreamInfo(ctx context.Context, streamName string) (client.StreamInfo, error)
	DeleteStream(ctx context.Context, streamName string) error
	ListEvents(ctx context.Context, streamName string) ([]client.StreamEventInfo, error)
	DeleteEvent(ctx context.Context, streamName, id string) error
	DeleteConsumer(ctx context.Context, streamName, group, consumer string) error
}

type APIServiceImpl struct {
	client client.Client
}

const streamMaxLen int64 = 1000

func NewAPIService(client client.Client) *APIServiceImpl {
	return &APIServiceImpl{
		client: client,
	}
}

func (s *APIServiceImpl) CreateNewStream(ctx context.Context, streamName string) error {
	id, err := s.client.CreateNewStream(ctx, streamName, streamMaxLen)
	if err != nil {
		return err
	}

	if err := s.client.DeleteEvent(ctx, streamName, id); err != nil {
		return err
	}

	return nil
}

func (s *APIServiceImpl) ListStreams(ctx context.Context) ([]client.StreamListItem, string, error) {
	streams, err := s.client.ListAllStreams(ctx)
	if err != nil {
		return nil, "", err
	}

	connString := s.client.GetRedisConnectionString()

	return streams, connString, nil
}

func (s *APIServiceImpl) GetStreamInfo(ctx context.Context, streamName string) (client.StreamInfo, error) {
	stream, err := s.client.GetFullStreamInfo(ctx, streamName)
	if err != nil {
		return client.StreamInfo{}, err
	}

	return stream, nil
}

func (s *APIServiceImpl) DeleteStream(ctx context.Context, streamName string) error {
	if err := s.client.WipeAckMeta(ctx, streamName); err != nil {
		return err
	}

	b64Stream := utils.Base64Encode(streamName)
	pattern := fmt.Sprintf("consumer:health:%s:*", b64Stream)
	if err := s.client.DeleteKeyWithPattern(ctx, pattern); err != nil {
		return err
	}

	return nil
}

func (s *APIServiceImpl) ListEvents(ctx context.Context, streamName string) ([]client.StreamEventInfo, error) {
	events, err := s.client.GetAllEvents(ctx, streamName)
	if err != nil {
		return nil, err
	}

	log.Printf("events: %+v", events)
	return events, nil
}

func (s *APIServiceImpl) DeleteEvent(ctx context.Context, streamName, id string) error {
	if err := s.client.DeleteEvent(ctx, streamName, id); err != nil {
		return err
	}

	if err := s.client.DeleteAckMeta(ctx, streamName, id); err != nil {
		return err
	}

	return nil
}

func (s *APIServiceImpl) DeleteConsumer(ctx context.Context, streamName, group, consumer string) error {
	if err := s.client.DeleteConsumer(ctx, streamName, group, consumer); err != nil {
		return err
	}
	return nil
}
