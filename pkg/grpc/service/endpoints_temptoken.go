package service

import (
	"context"
	"time"

	api_types "github.com/influenzanet/go-utils/pkg/api_types"
	"github.com/influenzanet/user-management-service/pkg/api"
	"github.com/influenzanet/user-management-service/pkg/models"
	"github.com/influenzanet/user-management-service/pkg/tokens"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const deleteTempTokensMinInterval = 10 * 60

var (
	lastTempTokenDeleteTime int64
)

func (s *userManagementServer) GetOrCreateTemptoken(ctx context.Context, t *api_types.TempTokenInfo) (*api.TempToken, error) {
	if t == nil || t.Purpose == "" || t.UserId == "" || t.InstanceId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument")
	}

	// Cleanup temptokens if this was not done recently:
	now := time.Now().Unix()
	if lastTempTokenDeleteTime+deleteTempTokensMinInterval < now {
		go s.CleanExpiredTemptokens(3600)
		lastTempTokenDeleteTime = now
	}

	tList, err := s.globalDBService.GetTempTokenForUser(t.InstanceId, t.UserId, t.Purpose)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &api.TempToken{}

	if len(tList) < 1 {
		tempToken := models.TempToken{
			UserID:     t.UserId,
			InstanceID: t.InstanceId,
			Purpose:    t.Purpose,
			Info:       t.Info,
			Expiration: t.Expiration,
		}

		if tempToken.Expiration == 0 {
			tempToken.Expiration = tokens.GetExpirationTime(time.Hour * 24 * 10)
		}

		token, err := s.globalDBService.AddTempToken(tempToken)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		resp.Token = token
	} else {
		resp.Token = tList[0].Token
	}
	return resp, nil
}

func (s *userManagementServer) GenerateTempToken(ctx context.Context, t *api_types.TempTokenInfo) (*api.TempToken, error) {
	if t == nil || t.Purpose == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument")
	}

	// Cleanup temptokens if this was not done recently:
	now := time.Now().Unix()
	if lastTempTokenDeleteTime+deleteTempTokensMinInterval < now {
		go s.CleanExpiredTemptokens(3600)
		lastTempTokenDeleteTime = now
	}

	tempToken := models.TempToken{
		UserID:     t.UserId,
		InstanceID: t.InstanceId,
		Purpose:    t.Purpose,
		Info:       t.Info,
		Expiration: t.Expiration,
	}

	if tempToken.Expiration == 0 {
		tempToken.Expiration = tokens.GetExpirationTime(time.Hour * 24 * 10)
	}

	token, err := s.globalDBService.AddTempToken(tempToken)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.TempToken{
		Token: token,
	}, nil
}

func (s *userManagementServer) GetTempTokens(ctx context.Context, t *api_types.TempTokenInfo) (*api_types.TempTokenInfos, error) {
	if t == nil || t.UserId == "" || t.InstanceId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument")
	}

	tokens, err := s.globalDBService.GetTempTokenForUser(t.InstanceId, t.UserId, t.Purpose)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return tokens.ToAPI(), nil
}

func (s *userManagementServer) DeleteTempToken(ctx context.Context, t *api.TempToken) (*api.ServiceStatus, error) {
	if t == nil || t.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument")
	}
	if err := s.globalDBService.DeleteTempToken(t.Token); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.ServiceStatus{
		Status:  api.ServiceStatus_NORMAL,
		Msg:     "deleted",
		Version: apiVersion,
	}, nil
}

func (s *userManagementServer) PurgeUserTempTokens(ctx context.Context, t *api_types.TempTokenInfo) (*api.ServiceStatus, error) {
	if t == nil || t.UserId == "" || t.InstanceId == "" {
		return nil, status.Error(codes.InvalidArgument, "missing argument")
	}
	if err := s.globalDBService.DeleteAllTempTokenForUser(t.InstanceId, t.UserId, t.Purpose); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.ServiceStatus{
		Status:  api.ServiceStatus_NORMAL,
		Msg:     "deleted",
		Version: apiVersion,
	}, nil
}
