package model

type Artifact struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SHA256      string `json:"sha256,omitempty"`
	DataBase64  string `json:"data_base64,omitempty"`
}

type EmailScanRequest struct {
	MessageID   string     `json:"message_id"`
	Sender      string     `json:"sender"`
	Subject     string     `json:"subject"`
	BodyPreview string     `json:"body_preview,omitempty"`
	URLs        []string   `json:"urls,omitempty"`
	Attachments []Artifact `json:"attachments,omitempty"`
}

type Finding struct{ Engine, Severity, Summary string }
type Verdict struct {
	Status   string    `json:"status"`
	Score    int       `json:"score"`
	Findings []Finding `json:"findings"`
}
