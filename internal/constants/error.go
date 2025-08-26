package constants

import "errors"

var (
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
)
