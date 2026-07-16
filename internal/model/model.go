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

	// SenderDisplay is the From header display name (e.g. "資安研發部-李沛宸").
	SenderDisplay string `json:"sender_display,omitempty"`
	// ReplyTo is the Reply-To address, if it differs from Sender.
	ReplyTo string `json:"reply_to,omitempty"`
	// AuthResults is the raw Authentication-Results header the client read from
	// the message (SPF/DKIM/DMARC). The API surfaces it; it does not perform DNS.
	AuthResults string `json:"auth_results,omitempty"`
	// Direction is "inbound" (default) or "outbound". Outbound enables DLP.
	Direction string `json:"direction,omitempty"`
	// QRURLs are URLs a client decoded from QR images when the server cannot
	// decode the image itself (e.g. QR embedded in a PDF the client rasterized).
	QRURLs []string `json:"qr_urls,omitempty"`
	// ReceivedChain is the ordered list of Received: header lines (top = most
	// recent hop) the client read from the message, for route forensics.
	ReceivedChain []string `json:"received,omitempty"`
}

type Finding struct {
	Engine   string `json:"engine"`
	Category string `json:"category,omitempty"`
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
}

type Verdict struct {
	Status   string    `json:"status"`
	Score    int       `json:"score"`
	Findings []Finding `json:"findings"`
}

// APIError is returned for 4xx validation and decode failures.
type APIError struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
