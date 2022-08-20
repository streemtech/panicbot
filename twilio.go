package panicbot

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type Twilio interface {
	SendMessage(toNumber, body string) error
}
type TwilioImpl struct {
	accountSID        string
	apiKey            string
	apiSecret         string
	twilioPhoneNumber string
	logger            *log.Logger
	client            *twilio.RestClient
}
type TwilioImplArgs struct {
	AccountSID        string
	APIKey            string
	APISecret         string
	TwilioPhoneNumber string
	Logger            *log.Logger
	Client            *twilio.RestClient
}

var _ Twilio = (*TwilioImpl)(nil)

func (Twilio *TwilioImpl) SendMessage(toNumber, body string) error {
	params := &api.CreateMessageParams{}
	params.SetTo(toNumber)
	params.SetFrom(Twilio.twilioPhoneNumber)
	params.SetBody(body)

	resp, err := Twilio.client.Api.CreateMessage(params)
	if err != nil {
		Twilio.logger.Errorf("Error: %s", err.Error())
		return fmt.Errorf("failed to send message: %s from %s to %s", body, Twilio.twilioPhoneNumber, toNumber)
	}

	Twilio.logger.Debugf("Message Sid: %s", *resp.Sid)
	return nil
}
func NewTwilio(args *TwilioImplArgs) (*TwilioImpl, error) {
	if args.AccountSID == "" {
		return nil, fmt.Errorf("AccountSID cannot be empty. Did you forget to set it in the config?")
	}

	if args.APIKey == "" {
		return nil, fmt.Errorf("APIKey cannot be empty. Did you forget to set it in the config?")
	}
	if args.APISecret == "" {
		return nil, fmt.Errorf("APISecret cannot be empty. Did you forget to set it in the config?")
	}

	if args.TwilioPhoneNumber == "" {
		return nil, fmt.Errorf("TwilioPhoneNumber cannot be empty. Did you forget to set it in the config?")
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username:   args.APIKey,
		Password:   args.APISecret,
		AccountSid: args.AccountSID,
	})
	twilioImpl := &TwilioImpl{
		accountSID:        args.AccountSID,
		apiKey:            args.APIKey,
		apiSecret:         args.APISecret,
		twilioPhoneNumber: args.TwilioPhoneNumber,
		client:            client,
		logger:            args.Logger,
	}
	return twilioImpl, nil
}
