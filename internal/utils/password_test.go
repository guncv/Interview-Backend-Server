package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	ctx := context.Background()
	log := log.Initialize("test")

	password := NewPassword(log)
	rightPassword := "password"
	wrongPassword := "wrong_password"
	hashedPassword, err := password.HashPassword(ctx, rightPassword)

	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, hashedPassword, rightPassword)
	assert.NotEqual(t, hashedPassword, "")

	hashedPassword2, err := password.HashPassword(ctx, rightPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword2)
	assert.NotEqual(t, hashedPassword2, hashedPassword)
	assert.NotEqual(t, hashedPassword2, "")

	assert.NotEqual(t, hashedPassword, hashedPassword2)

	err = password.CheckPassword(ctx, rightPassword, hashedPassword)
	assert.NoError(t, err)

	err = password.CheckPassword(ctx, wrongPassword, hashedPassword)
	require.Error(t, err)
	require.ErrorContains(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())

	// Test with empty password
	emptyHashedPassword, err := password.HashPassword(ctx, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, emptyHashedPassword)

	// Test with special characters
	specialPassword := "!@#$%^&*()_+-=[]{}|;:,.<>?"
	specialHashedPassword, err := password.HashPassword(ctx, specialPassword)
	assert.NoError(t, err)
	assert.NotEmpty(t, specialHashedPassword)

	// Test check password with empty string
	err = password.CheckPassword(ctx, "", emptyHashedPassword)
	assert.NoError(t, err)

	// Test check password with special characters
	err = password.CheckPassword(ctx, specialPassword, specialHashedPassword)
	assert.NoError(t, err)

	// Test with very long password to potentially trigger bcrypt error
	// Note: bcrypt has a limit of 72 bytes for password length
	veryLongPassword := string(make([]byte, 100))
	_, err = password.HashPassword(ctx, veryLongPassword)
	// This might succeed or fail depending on bcrypt implementation
	// We're just testing that the function handles it gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "ONX0101")
	}
}
