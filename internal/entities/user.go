package entities

import (
	"database/sql"
	"time"

	"gitlab.com/interview-simulation/interview-backend-server/internal/constants"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

type SignUpUserRequest struct {
	Email       string    `json:"email" validate:"required,email,max=100"`
	Password    string    `json:"password" validate:"required,min=8,max=100"`
	FullName    string    `json:"full_name" validate:"required,min=2,max=50"`
	PhoneNumber string    `json:"phone_number" validate:"required,min=10,max=20"`
	Country     string    `json:"country" validate:"required,min=2,max=50"`
	City        string    `json:"city" validate:"required,min=2,max=50"`
	Address     string    `json:"address" validate:"required,min=5,max=200"`
	PostalCode  string    `json:"postal_code" validate:"required,min=5,max=20"`
	Gender      string    `json:"gender" validate:"required,oneof=male female other"`
	DateOfBirth time.Time `json:"date_of_birth" validate:"required,date"`
}

type SignUpUserResponse struct {
	TokenId string `json:"token_id"`
}

type AdminCreateUserRequest struct {
	Email          string             `json:"email" validate:"required,email,max=100"`
	Password       string             `json:"password" validate:"required,min=8,max=100"`
	Name           string             `json:"name" validate:"required,min=2,max=50"`
	OrganizationID string             `json:"organization_id" validate:"required,uuid"`
	Role           constants.UserRole `json:"role" validate:"required,oneof=admin mentor trainee"`
}

type AdminCreateUserResponse struct {
	ID          string `json:"id"`
	AccessToken string `json:"access_token"`
}

type SignInByEmailAndPasswordRequest struct {
	Email    string `json:"email" validate:"required,email,max=100"`
	Password string `json:"password" validate:"required,min=1,max=100"`
}

type SignInResponse struct {
	ID             string `json:"id"`
	AccessToken    string `json:"access_token"`
	IsTempPassword bool   `json:"is_temp_password"`
}

type SignInServiceResponse struct {
	ID             string `json:"id"`
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	IsTempPassword bool   `json:"is_temp_password"`
}

type TokenRequest struct {
	UserID         string             `json:"user_id"`
	OrganizationID string             `json:"organization_id"`
	Role           constants.UserRole `json:"role"`
	Duration       time.Duration      `json:"duration"`
}

type CookieRequest struct {
	RefreshToken string        `json:"refresh_token"`
	Duration     time.Duration `json:"duration"`
	Domain       string        `json:"domain"`
}

type ResetUserPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

type SetInitPasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=100"`
}

type SetInitPasswordResponse struct {
	ID          string `json:"id"`
	AccessToken string `json:"access_token"`
}

type ForgotUserPasswordRequest struct {
	Email string `json:"email" validate:"required,email,max=100"`
}

type ForgotUserPasswordResponse struct {
	ID string `json:"id"`
}

type CreateOrganizationRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=100"`
	Address      string  `json:"address" validate:"required,min=5,max=200"`
	ContactEmail string  `json:"contact_email" validate:"required,email,max=100"`
	ContactPhone string  `json:"contact_phone" validate:"required,min=10,max=20"`
	ContactName  *string `json:"contact_name" validate:"omitempty,min=2,max=50"`
	WebsiteURL   *string `json:"website_url" validate:"omitempty,url,max=200"`
	Description  *string `json:"description" validate:"omitempty,max=500"`
	Country      string  `json:"country" validate:"required,len=2"`
}

type CreateOrganizationResponse struct {
	ID string `json:"id"`
}

type ListAllOrganizationsResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	ContactName  string `json:"contact_name"`
	WebsiteURL   string `json:"website_url"`
	Description  string `json:"description"`
	Country      string `json:"country"`
}

type ListOrganizationsFromCourseIdRequest struct {
	CourseID string `json:"course_id" validate:"required,uuid"`
}

type ListOrganizationsFromCourseIdResponse struct {
	ID                       string         `json:"id"`
	OrganizationID           string         `json:"organization_id"`
	CourseID                 string         `json:"course_id"`
	OrganizationName         string         `json:"organization_name"`
	OrganizationAddress      string         `json:"organization_address"`
	OrganizationContactEmail string         `json:"organization_contact_email"`
	OrganizationContactPhone string         `json:"organization_contact_phone"`
	OrganizationContactName  sql.NullString `json:"organization_contact_name"`
	OrganizationCountry      string         `json:"organization_country"`
	GivePermissionBy         string         `json:"give_permission_by"`
}
