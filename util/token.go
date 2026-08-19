package util

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func CreateToken(claims jwt.MapClaims, key string, tmout time.Duration) (string, error) {
	claims["times"] = time.Now()
	if tmout > 0 {
		claims["timeout"] = time.Now().Add(tmout)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokens, err := token.SignedString([]byte(key))
	if err != nil {
		return "", fmt.Errorf("sign JWT token: %w", err)
	}
	return tokens, nil
}
func SetToken(c *gin.Context, p jwt.MapClaims, key string, rem bool, doman ...string) (string, error) {
	tmout := time.Hour * 5
	if rem {
		tmout = time.Hour * 24 * 5
	}
	tokens, err := CreateToken(p, key, tmout)
	if err != nil {
		return "", fmt.Errorf("create token for cookie: %w", err)
	}
	cke := http.Cookie{ // #nosec G124 -- cookie attributes are configurable at runtime
		Name:     "gokinstk",
		Value:    tokens,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	if len(doman) > 0 {
		cke.Domain = doman[0]
	}

	cke.MaxAge = 60 * 60 * 5
	if rem {
		cke.MaxAge = 60 * 60 * 24 * 5
	}
	c.Writer.Header().Add("Set-Cookie", cke.String())
	return tokens, nil
}

func ClearToken(c *gin.Context, doman ...string) error {
	cke := http.Cookie{ // #nosec G124 -- cookie attributes are configurable at runtime
		Name:     "gokinstk",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	if len(doman) > 0 {
		cke.Domain = doman[0]
	}
	cke.MaxAge = -1
	c.Writer.Header().Set("Set-Cookie", cke.String())
	return nil
}

func getToken(c *gin.Context) string {
	tkc, err := c.Request.Cookie("gokinstk")
	if err != nil {
		return ""
	}
	return tkc.Value
}
func getTokenAuth(c *gin.Context) string {
	ats := c.GetHeader("Authorization")
	if ats == "" {
		return ""
	}
	aths, err := url.PathUnescape(ats)
	if err != nil {
		return ""
	}
	aths = strings.TrimPrefix(aths, "TOKEN ")
	return aths
}
func GetTokens(s string, key string) jwt.MapClaims {
	if s == "" {
		return nil
	}
	token, err := jwt.Parse(s, func(token *jwt.Token) (any, error) {
		// Validate the signing method to prevent algorithm confusion attacks.
		// We only accept HMAC-based signing (HS512) since that's what CreateToken uses.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(key), nil
	})
	if err == nil {
		claim, ok := token.Claims.(jwt.MapClaims)
		if ok {
			return claim
		}
	}
	return nil
}
func GetToken(c *gin.Context, key string) jwt.MapClaims {
	tk := getTokenAuth(c)
	if tk == "" {
		tk = getToken(c)
	}
	if tk == "" {
		tk = c.Query("authToken")
	}
	return GetTokens(tk, key)
}
