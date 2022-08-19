package panicbot

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type Twilio interface {
	SendMessage(phoneNumber, message string) error
}
type TwilioImpl struct {
	accountSID        string
	authToken         string
	twilioPhoneNumber string
	logger            *log.Logger
	sendMessage       func(phoneNumber, message string)
}
type TwilioImplArgs struct {
	AccountSID        string
	AuthToken         string
	TwilioPhoneNumber string
	Logger            *log.Logger
	SendMessage       func(phoneNumber, message string)
}

var _ Twilio = (*TwilioImpl)(nil)

func (Twilio *TwilioImpl) SendMessage(phoneNumber, message string) error {
	return nil
}
func NewTwilio(args *TwilioImplArgs) (*TwilioImpl, error) {
	if args.AccountSID == "" {
		return nil, fmt.Errorf("AccountSID cannot be empty. Did you forget to set it in the config?")
	}

	if args.AuthToken == "" {
		return nil, fmt.Errorf("AuthToken cannot be empty. Did you forget to set it in the config?")
	}

	if args.TwilioPhoneNumber == "" {
		return nil, fmt.Errorf("TwilioPhoneNumber cannot be empty. Did you forget to set it in the config?")
	}

	twilioImpl := &TwilioImpl{
		accountSID:        args.AccountSID,
		authToken:         args.AuthToken,
		twilioPhoneNumber: args.TwilioPhoneNumber,
	}
	return twilioImpl, nil
}
