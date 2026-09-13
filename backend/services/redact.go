package services

import (
	"errors"
	"regexp"
)

// botTokenPattern matches the credential inside a Telegram Bot API URL, e.g.
// "https://api.telegram.org/bot123456789:AAF…/getMe".
var botTokenPattern = regexp.MustCompile(`bot\d{5,}:[A-Za-z0-9_-]+`)

// RedactSecrets removes Telegram bot tokens from a message.
//
// net/http reports a transport failure with the full request URL, and a bot
// token is part of that URL, so an unredacted error prints the credential into
// the status card, the server log, and every screenshot of either. Anything
// that may carry a transport error goes through here on the way out.
func RedactSecrets(message string) string {
	if message == "" {
		return message
	}
	return botTokenPattern.ReplaceAllString(message, "bot<redacted>")
}

// RedactError is RedactSecrets for an error value.
func RedactError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(RedactSecrets(err.Error()))
}
