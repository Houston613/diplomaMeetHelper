package integration_test

import (
	"errors"
	"testing"
	"time"

	"diplomaMeetHelper/internal/adapters/db/postgres"
	"diplomaMeetHelper/internal/domain"

	"github.com/google/uuid"
)

func TestMeetingRepository_FullPipeline_Integration(t *testing.T) {
	pool, ctx := setupTestDB(t)
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	meetingRepo := postgres.NewMeetingRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)

	user1 := "pipeline-user-1-" + uuid.NewString()
	user2 := "pipeline-user-2-" + uuid.NewString()

	_, _, err := userRepo.EnsureUser(ctx, user1)
	if err != nil {
		t.Fatalf("failed to ensure user 1: %v", err)
	}
	_, _, err = userRepo.EnsureUser(ctx, user2)
	if err != nil {
		t.Fatalf("failed to ensure user 2: %v", err)
	}

	meetingID := uuid.New()
	jobID := uuid.New()
	now := time.Now()

	meeting := &domain.Meeting{
		ID:        meetingID,
		UserID:    user1,
		Filename:  "standup.mp3",
		FilePath:  "/tmp/standup.mp3",
		Status:    domain.StatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	job := &domain.ProcessingJob{
		ID:           jobID,
		MeetingID:    meetingID,
		UserID:       user1,
		Status:       domain.StatusCreated,
		ErrorMessage: "",
		RetryCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 1. Create meeting with job
	if err := meetingRepo.CreateMeetingWithJob(ctx, meeting, job); err != nil {
		t.Fatalf("CreateMeetingWithJob failed: %v", err)
	}

	// 2. Complete job with results
	transcript := "Meeting transcription text about sprint planning."
	summary := "Sprint planning summary: 5 story points planned."
	if err := jobRepo.CompleteJobWithResults(ctx, jobID, meetingID, transcript, summary); err != nil {
		t.Fatalf("CompleteJobWithResults failed: %v", err)
	}

	// 3. Get Meeting Details (Owner)
	details, err := meetingRepo.GetMeetingDetails(ctx, meetingID, user1)
	if err != nil {
		t.Fatalf("GetMeetingDetails failed: %v", err)
	}
	if details.Meeting.Status != domain.StatusCompleted {
		t.Errorf("expected meeting status 'completed', got %s", details.Meeting.Status)
	}
	if details.Transcript == nil || details.Transcript.Content != transcript {
		t.Errorf("expected transcript content %q, got %+v", transcript, details.Transcript)
	}
	if details.Summary == nil || details.Summary.Content != summary {
		t.Errorf("expected summary content %q, got %+v", summary, details.Summary)
	}

	// 4. User Isolation Check (User 2 should be denied access to User 1's meeting)
	_, err = meetingRepo.GetMeetingDetails(ctx, meetingID, user2)
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied for other user, got: %v", err)
	}

	// 5. List Meetings
	list, err := meetingRepo.ListMeetings(ctx, user1)
	if err != nil {
		t.Fatalf("ListMeetings failed: %v", err)
	}
	if len(list) == 0 || list[0].ID != meetingID {
		t.Errorf("expected list to contain meeting %s", meetingID)
	}
}
