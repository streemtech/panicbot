package panicbot

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type Twilio interface {
	SendMessage(toNumber, fromNumber, body string) error
}
type TwilioImpl struct {
	accountSID        string
	authToken         string
	twilioPhoneNumber string
	logger            *log.Logger
	sendMessage       func(toNumber, fromNumber, body string)
	client            *twilio.RestClient
}
type TwilioImplArgs struct {
	AccountSID        string
	AuthToken         string
	TwilioPhoneNumber string
	Logger            *log.Logger
	SendMessage       func(toNumber, fromNumber, body string)
	Client            *twilio.RestClient
}

var _ Twilio = (*TwilioImpl)(nil)

func (Twilio *TwilioImpl) SendMessage(toNumber, fromNumber, body string) error {
	params := &api.CreateMessageParams{}
	params.SetTo(toNumber)
	params.SetFrom(fromNumber)
	params.SetBody(body)

	resp, err := Twilio.client.Api.CreateMessage(params)
	if err != nil {
		Twilio.logger.Errorf("Error: %s", err.Error())
		return fmt.Errorf("failed to send message: %s from %s to %s", body, fromNumber, toNumber)
	}

	Twilio.logger.Info("Message Sid: " + *resp.Sid)
	return nil
}
func NewTwilio(args *TwilioImplArgs) (*TwilioImpl, error) {
	if args.AccountSID == "" {
		return nil, fmt.Errorf("AccountSID cannot be empty. Did you forget to set it in the config?")
	}

	if args.TwilioPhoneNumber == "" {
		return nil, fmt.Errorf("TwilioPhoneNumber cannot be empty. Did you forget to set it in the config?")
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		// Username:   args.APIKey,
		// Password:   args.APISecret,
		Username: args.AccountSID,
		Password: args.AuthToken,
	})
	twilioImpl := &TwilioImpl{
		accountSID:        args.AccountSID,
		twilioPhoneNumber: args.TwilioPhoneNumber,
		client:            client,
		logger:            args.Logger,
	}
	return twilioImpl, nil
}
