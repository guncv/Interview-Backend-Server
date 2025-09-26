package constants

import "errors"

var (
	// General
	ErrPermissionDenied = errors.New("permission denied")

	ErrCategoryIDRequired           = errors.New("category id is required")
	ErrExpiredToken                 = errors.New("token has expired")
	ErrInvalidToken                 = errors.New("token is invalid")
	ErrInvalidRole                  = errors.New("invalid role")
	ErrCategoryNotFound             = errors.New("category not found")
	ErrCourseNotFound               = errors.New("course not found")
	ErrCourseAlreadyExists          = errors.New("course already exists")
	ErrOrganizationNotFound         = errors.New("organization not found")
	ErrCourseSectionNotFound        = errors.New("course section not found")
	ErrCourseSectionAlreadyExists   = errors.New("course section already exists")
	ErrCourseSectionInvalidRequest  = errors.New("course section invalid request")
	ErrSectionContentNotFound       = errors.New("section content not found")
	ErrSectionContentAlreadyExists  = errors.New("section content already exists")
	ErrSectionContentInvalidRequest = errors.New("section content invalid request")

	// Interview Session
	ErrInterviewSessionStartEndTimeNotFound = errors.New("interview session start end time not found")
	ErrInterviewSessionStartTimeNotFound    = errors.New("interview session start time not found")
	ErrInterviewSessionEndTimeNotFound      = errors.New("interview session end time not found")
	ErrInterviewSessionInvalidOverallScore  = errors.New("interview session overall score is invalid")
	ErrInterviewSessionInvalidStatus        = errors.New("interview session status is invalid")
	ErrInterviewSessionNotStarted           = errors.New("interview session is not started")
	ErrInterviewSessionTurnNoNegative       = errors.New("interview session turn no cannot be negative")
	ErrInterviewSessionNotFound             = errors.New("interview session not found")

	// Issue Reports
	ErrIssueReportNotOpen      = errors.New("issue report not open")
	ErrIssueReportUnauthorized = errors.New("this user is not the owner of the issue report")
	ErrIssueReportIDRequired   = errors.New("issue report ID is required")

	// Resume
	ErrResumeDoesNotBelongToUser = errors.New("resume does not belong to user")
)
