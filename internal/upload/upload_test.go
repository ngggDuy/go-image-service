package upload

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestProcess(t *testing.T) {
	const (
		filename = "photo.jpg"
		ext      = ".jpg"
	)
	data := []byte("original-bytes")

	// Each case declares the calls it expects. gomock fails the test if an
	// expected call never happens, or if the code makes an unexpected call — so
	// the failure cases also prove Process stops at the first error.
	tests := []struct {
		name    string
		setup   func(s *MockStore, repo *MockRepository, starter *MockUploadStarter)
		wantErr bool
	}{
		{
			name: "success: saves original, records processing, starts workflow",
			setup: func(s *MockStore, repo *MockRepository, starter *MockUploadStarter) {
				// The id is random, so match it with gomock.Any().
				s.EXPECT().Save(gomock.Any(), "original", ext, data).Return(nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, ext, "processing").Return(nil)
				starter.EXPECT().Start(gomock.Any(), gomock.Any(), ext, filename).Return(nil)
			},
		},
		{
			name: "stops when saving the original fails",
			setup: func(s *MockStore, repo *MockRepository, starter *MockUploadStarter) {
				s.EXPECT().Save(gomock.Any(), "original", ext, data).Return(errors.New("disk full"))
			},
			wantErr: true,
		},
		{
			name: "stops when recording metadata fails",
			setup: func(s *MockStore, repo *MockRepository, starter *MockUploadStarter) {
				s.EXPECT().Save(gomock.Any(), "original", ext, data).Return(nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, ext, "processing").Return(errors.New("db down"))
			},
			wantErr: true,
		},
		{
			name: "stops when starting the workflow fails",
			setup: func(s *MockStore, repo *MockRepository, starter *MockUploadStarter) {
				s.EXPECT().Save(gomock.Any(), "original", ext, data).Return(nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any(), filename, ext, "processing").Return(nil)
				starter.EXPECT().Start(gomock.Any(), gomock.Any(), ext, filename).Return(errors.New("temporal down"))
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			s := NewMockStore(ctrl)
			repo := NewMockRepository(ctrl)
			starter := NewMockUploadStarter(ctrl)
			tc.setup(s, repo, starter)

			id, err := New(s, repo, starter).Process(context.Background(), data, filename, ext)

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
				t.Errorf("returned id length = %d, want 32", len(id))
			}
		})
	}
}
