package service

import (
	"context"
	"errors"
	"event-manager/service/client"
	"testing"
)

type fakeClient struct {
	listEventsCalls int
	listEventsErr   error
}

func (f *fakeClient) GetRedisConnectionString() string { return "" }

func (f *fakeClient) CreateNewStream(context.Context, string, int64) (string, error) { return "", nil }

func (f *fakeClient) ListAllStreams(context.Context) ([]client.StreamListItem, error) {
	return nil, nil
}

func (f *fakeClient) GetFullStreamInfo(context.Context, string) (client.StreamInfo, error) {
	return client.StreamInfo{}, nil
}

func (f *fakeClient) WipeAckMeta(context.Context, string) error { return nil }

func (f *fakeClient) DeleteKeyWithPattern(context.Context, string) error { return nil }

func (f *fakeClient) DeleteEvent(context.Context, string, ...string) error { return nil }

func (f *fakeClient) GetAllEvents(context.Context, string) ([]client.StreamEventInfo, error) {
	f.listEventsCalls++
	return nil, f.listEventsErr
}

func (f *fakeClient) DeleteAckMeta(context.Context, string, string) error { return nil }

func (f *fakeClient) DeleteConsumer(context.Context, string, string, string) error { return nil }

func TestListEventsDelegatesToClient(t *testing.T) {
	wantErr := errors.New("redis failed")
	fake := &fakeClient{listEventsErr: wantErr}
	svc := NewAPIService(fake)

	_, err := svc.ListEvents(context.Background(), "orders")

	if !errors.Is(err, wantErr) {
		t.Fatalf("ListEvents error = %v, want %v", err, wantErr)
	}
	if fake.listEventsCalls != 1 {
		t.Fatalf("GetAllEvents calls = %d, want 1", fake.listEventsCalls)
	}
}
