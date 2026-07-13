package documents

import (
	"context"
)

type OCRInput struct {
	FilePath     string `json:"file_path"`
	Language     string `json:"language"`
	OutputFormat string `json:"output_format"`
}

type OCRResult struct {
	Text        string                   `json:"text"`
	Pages       []OCRPage                `json:"pages"`
	Provider    string                   `json:"provider"`
	RawResponse interface{}              `json:"raw_response"`
}

type OCRPage struct {
	PageNumber int                      `json:"page_number"`
	Language   string                   `json:"language"`
	Text       string                   `json:"text"`
	Blocks     []interface{}            `json:"blocks"`
	Confidence float64                  `json:"confidence"`
}

type OCRProvider interface {
	ExtractDocument(ctx context.Context, input OCRInput) (*OCRResult, error)
}

type SarvamOCRProvider struct {
	APIKey string
}

func (p *SarvamOCRProvider) ExtractDocument(ctx context.Context, input OCRInput) (*OCRResult, error) {
	// Delegate/mock implementation on the orchestrator gateway.
	// In the BimaNyaya architecture, the Go API orchestrates calls 
	// by invoking the Python FastAPI worker pool, which executes the actual ML pipelines.
	return &OCRResult{
		Provider: "Sarvam",
	}, nil
}

type FallbackOCRProvider struct{}

func (p *FallbackOCRProvider) ExtractDocument(ctx context.Context, input OCRInput) (*OCRResult, error) {
	return &OCRResult{
		Provider: "Fallback",
	}, nil
}

type MockOCRProvider struct{}

func (p *MockOCRProvider) ExtractDocument(ctx context.Context, input OCRInput) (*OCRResult, error) {
	return &OCRResult{
		Provider: "Mock",
	}, nil
}
