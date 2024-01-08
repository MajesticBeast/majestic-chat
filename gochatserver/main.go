package main

import (
	"context"
	"github.com/majesticbeast/gochat/gochatserver/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

type chatServer struct {
	gochat.UnimplementedChatServiceServer
	clients map[string]gochat.ChatService_JoinChatServer
	mu      sync.Mutex
}

func (s *chatServer) SendMessage(ctx context.Context, req *gochat.Message) (*gochat.MessageAck, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	senderId := req.GetClientId()
	content := req.GetContent()
	timestamp := req.GetTimestamp()
	username := req.GetUsername()

	for _, client := range s.clients {
		if err := client.Send(&gochat.Message{
			Username:  username,
			ClientId:  senderId,
			Content:   content,
			Timestamp: timestamp,
		}); err != nil {
			log.Println(err)
			return nil, err
		}
	}

	log.Println("Received message to send: ", senderId, content, timestamp, username)

	return &gochat.MessageAck{Success: true}, nil
}

func (s *chatServer) JoinChat(req *gochat.JoinRequest, stream gochat.ChatService_JoinChatServer) error {
	s.mu.Lock()
	s.clients[req.ClientId] = stream
	s.mu.Unlock()

	<-stream.Context().Done()

	s.mu.Lock()
	delete(s.clients, req.ClientId)
	s.mu.Unlock()

	return nil
}

func main() {
	lis, err := net.Listen("tcp", ":3001")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	srv := &chatServer{
		clients: make(map[string]gochat.ChatService_JoinChatServer),
	}

	gochat.RegisterChatServiceServer(s, srv)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to gochatserver gRPC gochatserver: %v", err)
	}
}
