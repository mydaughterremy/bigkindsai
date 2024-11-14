package service

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

type ChatCompletionOptions struct {
	options azopenai.ChatCompletionsOptions
}

func NewLLMClient(provider string, modelName string) (*azopenai.Client, error) {
	switch provider {
	case "openai":
		keyCredential := azcore.NewKeyCredential(os.Getenv("UPSTAGE_OPENAI_KEY"))
		client, err := azopenai.NewClientForOpenAI("https://api.openai.com/v1", keyCredential, nil)
		if err != nil {
			return nil, err
		}
		return client, nil
	case "azure":
		keyCredential := azcore.NewKeyCredential(os.Getenv("UPSTAGE_AZURE_KEY"))
		client, err := azopenai.NewClientWithKeyCredential(fmt.Sprintf("https://%s.openai.azure.com", os.Getenv("UPSTAGE_AZURE_OPENAI_HOST")), keyCredential, nil)
		if err != nil {
			return nil, err
		}
		return client, nil
	default:
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
}

func newChatCompletionOptions() *ChatCompletionOptions {
	return &ChatCompletionOptions{
		options: azopenai.ChatCompletionsOptions{},
	}
}

func (s *ChatCompletionOptions) SetDeploymentName(deploymentName string) {
	s.options.DeploymentName = &deploymentName
}

func (s *ChatCompletionOptions) SetSeed(seedInt int64) {
	s.options.Seed = &seedInt
}

func (s *ChatCompletionOptions) SetTemperature(temperature float32) {
	s.options.Temperature = &temperature
}

func (s *ChatCompletionOptions) SetJsonResponseFormat() {
	s.options.ResponseFormat = &azopenai.ChatCompletionsJSONResponseFormat{}
}

func (s *ChatCompletionOptions) SetMessages(messages []azopenai.ChatRequestMessageClassification) {
	s.options.Messages = messages
}
