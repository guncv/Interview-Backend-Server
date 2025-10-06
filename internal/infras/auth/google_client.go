package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleUserInfo struct {
	ID            string `json:"sub"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type GoogleClient interface {
	ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error)
	GetAuthURL(state string) string
}

type googleClient struct {
	log       *log.Logger
	cfg       *config.Config
	cfgGoogle *oauth2.Config
}

func NewGoogleClient(logger *log.Logger, cfg *config.Config) GoogleClient {
	cfgGoogle := &oauth2.Config{
		ClientID:     cfg.OAuthConfig.OAuthGoogleClientID,
		ClientSecret: cfg.OAuthConfig.OAuthGoogleClientSecret,
		RedirectURL:  cfg.OAuthConfig.OAuthGoogleRedirectURI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	return &googleClient{
		log:       logger,
		cfg:       cfg,
		cfgGoogle: cfgGoogle,
	}
}

func (g *googleClient) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {

	token, err := g.cfgGoogle.Exchange(ctx, code)
	if err != nil {
		g.log.ErrorWithID(ctx, "[Google Client: ExchangeCodeForToken] Error exchanging auth code", err)
		return nil, err
	}
	return token, nil
}

func (g *googleClient) GetUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {

	client := g.cfgGoogle.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		g.log.ErrorWithID(ctx, "[Google Client: GetUserInfo] Error fetching user info", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		g.log.ErrorWithID(ctx, "[Google Client: GetUserInfo] Google API returned status ", resp.StatusCode)
		return nil, err
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		g.log.ErrorWithID(ctx, "[Google Client: GetUserInfo] Error decoding user info", err)
		return nil, err
	}

	if userInfo.ID == "" || userInfo.Email == "" {
		g.log.ErrorWithID(ctx, "[Google Client: GetUserInfo] Missing required Google user info fields")
		return nil, err
	}

	return &userInfo, nil
}

func (g *googleClient) GetAuthURL(state string) string {
	result := g.cfgGoogle.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return result
}
