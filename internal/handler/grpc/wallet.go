package grpchandler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/zgq/wallet/gen/wallet"
	"github.com/zgq/wallet/internal/domain"
	svc "github.com/zgq/wallet/internal/service"
)

// Server implements pb.WalletServiceServer.
type Server struct {
	pb.UnimplementedWalletServiceServer
	svc svc.Service
}

func NewServer(s svc.Service) *Server {
	return &Server{svc: s}
}

func (s *Server) CreateWallet(ctx context.Context, _ *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	w, err := s.svc.CreateWallet(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CreateWalletResponse{Id: w.ID, Balance: w.Balance}, nil
}

func (s *Server) GetWallet(ctx context.Context, req *pb.GetWalletRequest) (*pb.GetWalletResponse, error) {
	w, err := s.svc.GetWallet(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.GetWalletResponse{Id: w.ID, Balance: w.Balance}, nil
}

func (s *Server) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	err := s.svc.Transfer(ctx, req.GetFromId(), req.GetToId(), req.GetAmount())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, domain.ErrInsufficientFunds),
			errors.Is(err, domain.ErrInvalidAmount),
			errors.Is(err, domain.ErrSameWallet):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}
	return &pb.TransferResponse{}, nil
}
