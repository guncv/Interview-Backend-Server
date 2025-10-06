package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/interview-simulation/interview-backend-server/internal/config"
	"gitlab.com/interview-simulation/interview-backend-server/internal/infras/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

type FacebookUserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
	Locale string `json:"locale"`
}

type FacebookClient interface {
	ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error)
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*FacebookUserInfo, error)
	GetAuthURL(state string) string
}

type facebookClient struct {
	log         *log.Logger
	cfg         *config.Config
	cfgFacebook *oauth2.Config
}

func NewFacebookClient(logger *log.Logger, cfg *config.Config) FacebookClient {
	cfgFacebook := &oauth2.Config{
		ClientID:     cfg.OAuthConfig.OAuthFacebookClientID,
		ClientSecret: cfg.OAuthConfig.OAuthFacebookClientSecret,
		RedirectURL:  cfg.OAuthConfig.OAuthFacebookRedirectURI,
		Scopes:       []string{"email", "public_profile"},
		Endpoint:     facebook.Endpoint,
	}

	return &facebookClient{
		log:         logger,
		cfg:         cfg,
		cfgFacebook: cfgFacebook,
	}
}

func (f *facebookClient) ExchangeCodeForToken(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := f.cfgFacebook.Exchange(ctx, code)
	if err != nil {
		f.log.ErrorWithID(ctx, "[Facebook Client: ExchangeCodeForToken] Error exchanging auth code", err)
		return nil, err
	}
	return token, nil
}

func (f *facebookClient) GetUserInfo(ctx context.Context, token *oauth2.Token) (*FacebookUserInfo, error) {

	client := f.cfgFacebook.Client(ctx, token)
	url := fmt.Sprintf("https://graph.facebook.com/me?fields=id,name,email,picture,locale&access_token=%s", token.AccessToken)

	resp, err := client.Get(url)
	if err != nil {
		f.log.ErrorWithID(ctx, "[Facebook Client: GetUserInfo] Error fetching user info", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		f.log.ErrorWithID(ctx, "[Facebook Client: GetUserInfo] Facebook API returned status ", resp.StatusCode)
		return nil, fmt.Errorf("facebook API returned status %d", resp.StatusCode)
	}

	var userInfo FacebookUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		f.log.ErrorWithID(ctx, "[Facebook Client: GetUserInfo] Error decoding user info", err)
		return nil, err
	}

	if userInfo.ID == "" || userInfo.Email == "" {
		f.log.ErrorWithID(ctx, "[Facebook Client: GetUserInfo] Missing required Facebook user info fields")
		return nil, fmt.Errorf("missing required Facebook user info fields")
	}

	return &userInfo, nil
}

func (f *facebookClient) GetAuthURL(state string) string {
	result := f.cfgFacebook.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return result
}
