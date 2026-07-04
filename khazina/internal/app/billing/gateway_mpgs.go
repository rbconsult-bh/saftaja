package billing

import "context"

type mpgsCardGateway struct{}

func NewMPGSCardGateway() CardGateway {
	return &mpgsCardGateway{}
}

func (mcg *mpgsCardGateway) CreateSession(ctx context.Context, r CreateSessionRequest) (*CreateSessionResponse, error) {
	return &CreateSessionResponse{}, nil
}
