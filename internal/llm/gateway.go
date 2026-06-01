package llm

type Gateway interface {
	Chat(prompt string) (string, error)
}

type SimpleGateway struct {
	client *Client
}

func NewSimpleGateway(baseURL string) *SimpleGateway {
	return &SimpleGateway{
		client: NewClient(baseURL),
	}
}

func (g *SimpleGateway) Chat(prompt string) (string, error) {
	return g.client.Chat(prompt)
}
