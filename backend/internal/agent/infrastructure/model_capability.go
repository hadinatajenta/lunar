package infrastructure

import "lunar/backend/internal/agent/domain"

func (c *ChatClient) SupportsImageInput(modelOption string) bool {
	actualModel, _, _ := c.resolveModel(modelOption)
	return domain.ModelAcceptsImages(actualModel)
}
