package upload

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNegotiate(t *testing.T) {
	const (
		filename    = "photo.jpg"
		contentType = "image/jpeg"
		userID      = "user-1"
	)

	tests := []struct {
		name    string
		setup   func(repo *MockRepository, obj *MockPresigner, pipe *MockPipeline)
		wantErr bool
	}{
		{
			name: "success: creates pending row, starts workflow, presigns",
			setup: func(repo *MockRepository, obj *MockPresigner, pipe *MockPipeline) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, contentType, "pending", userID).Return(nil)
				pipe.EXPECT().Start(gomock.Any(), gomock.Any(), contentType, filename).Return(nil)
				obj.EXPECT().Presign(gomock.Any(), gomock.Any(), "PUT", gomock.Any()).Return("http://minio/presigned", nil)
			},
		},
		{
			name: "stops when the row can't be created",
			setup: func(repo *MockRepository, obj *MockPresigner, pipe *MockPipeline) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, contentType, "pending", userID).Return(errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "stops when the workflow can't start",
			setup: func(repo *MockRepository, obj *MockPresigner, pipe *MockPipeline) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, contentType, "pending", userID).Return(nil)
				pipe.EXPECT().Start(gomock.Any(), gomock.Any(), contentType, filename).Return(errors.New("temporal down"))
			},
			wantErr: true,
		},
		{
			name: "stops when presigning fails",
			setup: func(repo *MockRepository, obj *MockPresigner, pipe *MockPipeline) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, contentType, "pending", userID).Return(nil)
				pipe.EXPECT().Start(gomock.Any(), gomock.Any(), contentType, filename).Return(nil)
				obj.EXPECT().Presign(gomock.Any(), gomock.Any(), "PUT", gomock.Any()).Return("", errors.New("presign failed"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := NewMockRepository(ctrl)
			obj := NewMockPresigner(ctrl)
			pipe := NewMockPipeline(ctrl)
			tc.setup(repo, obj, pipe)

			id, url, err := New(repo, obj, pipe).Negotiate(context.Background(), filename, contentType, userID)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(id) != 32 {
				t.Errorf("id length = %d, want 32", len(id))
			}
			if url == "" {
				t.Error("expected a presigned url, got empty")
			}
		})
	}
}

func TestComplete_Signals(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockRepository(ctrl)
	obj := NewMockPresigner(ctrl)
	pipe := NewMockPipeline(ctrl)

	pipe.EXPECT().Signal(gomock.Any(), "abc123").Return(nil)

	if err := New(repo, obj, pipe).Complete(context.Background(), "abc123"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
}
