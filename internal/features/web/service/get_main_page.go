package web_service

import (
	"fmt"
)

const mainPageFilePath = "index.html"

func (s *WebService) GetMainPage() ([]byte, error) {
	html, err := s.webRepository.GetFile(mainPageFilePath)
	if err != nil {
		return nil, fmt.Errorf("get file from repository: %w", err)
	}
	return html, nil
}
