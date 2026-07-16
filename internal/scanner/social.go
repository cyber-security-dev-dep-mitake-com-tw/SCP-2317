package scanner

import (
	"sort"
	"strings"

	"github.com/example/scp2317-agent/internal/model"
)

// intentLexicon maps a social-engineering intent to trigger phrases. Phrases are
// deliberately specific (multi-word where possible) so a single ordinary word
// does not raise a flag. English + Traditional Chinese are covered.
var intentLexicon = map[string][]string{
	"urgency": {
		"urgent", "immediately", "as soon as possible", "within 24 hours",
		"right away", "account will be suspended", "will expire", "act now",
		"立即", "緊急", "儘快", "24小時內", "逾期", "帳戶將被凍結", "即將到期",
	},
	"authority": {
		"ceo", "cfo", "president", "on behalf of the director", "your manager",
		"executive", "board of directors",
		"董事長", "總經理", "執行長", "財務長", "主管交辦", "長官",
	},
	"financial": {
		"wire transfer", "bank transfer", "change of bank account",
		"update payment", "remittance", "gift card", "outstanding invoice",
		"payment details",
		"匯款", "轉帳", "變更匯款", "更改帳戶", "付款資訊", "禮品卡", "未付發票", "銀行帳號",
	},
	"credential": {
		"verify your account", "confirm your identity", "click here", "sign in",
		"reset your password", "log in to", "scan the qr", "download the image",
		"驗證您的帳戶", "確認身分", "點擊這裡", "重設密碼", "掃描 qr", "這裡下載圖片", "登入以",
	},
}

// socialEngine scores social-engineering / BEC intent from subject + body.
func (p *Pipeline) socialEngine(r model.EmailScanRequest) []model.Finding {
	text := strings.ToLower(r.Subject + "\n" + r.BodyPreview)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	hits := map[string]bool{}
	for intent, phrases := range intentLexicon {
		for _, ph := range phrases {
			if strings.Contains(text, ph) {
				hits[intent] = true
				break
			}
		}
	}
	if len(hits) < 2 {
		return nil
	}
	cats := make([]string, 0, len(hits))
	for k := range hits {
		cats = append(cats, k)
	}
	sort.Strings(cats)

	sev := "medium"
	// Classic BEC pressure pattern: a money ask wrapped in urgency or authority.
	if hits["financial"] && (hits["urgency"] || hits["authority"]) {
		sev = "high"
	}
	return []model.Finding{{
		Engine:   "social-engineering",
		Category: "social-engineering",
		Severity: sev,
		Summary:  "Social-engineering language detected (" + strings.Join(cats, ", ") + ")",
	}}
}
