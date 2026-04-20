package config

type Config struct {
	Brand          string
	CallbackURL    string
	RequiredScopes []string
}

func Default() Config {
	return Config{
		Brand:       "xfchat.iflytek.com",
		CallbackURL: "http://localhost:8080/callback",
		RequiredScopes: []string{
			"docs:document:readonly",
			"im:message:create_as_bot",
		},
	}
}
