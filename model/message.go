package model

const (
	MessageLenConstraint = 4096
)

// TelegramMessage contains whom to send and what to send
type TelegramMessage struct {
	ChatIds []int64
	Text    []string
	Photos  []string
	Gif     []string
	Doc     []string
}

// Extend add fields of other to this object
func (t *TelegramMessage) Extend(other *TelegramMessage) {
	t.Text = append(t.Text, other.Text...)
	t.Photos = append(t.Photos, other.Photos...)
}

// AddText add string with text
func (t *TelegramMessage) AddText(text string) {
	t.Text = append(t.Text, text)
}

// AddPhoto add url string to send
func (t *TelegramMessage) AddPhoto(url string) {
	if url != "" {
		t.Photos = append(t.Photos, url)
	}
}

// Normalize text len to be less constraint
func (t *TelegramMessage) Normalize() {
	text := t.Text
	t.Text = nil
	for _, msg := range text {
		runes := []rune(msg)
		for len(runes) != 0 {
			l := min(MessageLenConstraint, len(runes))
			t.AddText(string(runes[:l]))
			runes = runes[l:]
		}
	}
}

func (t *TelegramMessage) AddGif(url string) {
	if url != "" {
		t.Gif = append(t.Gif, url)
	}
}

func (t *TelegramMessage) AddFile(url string) {
	if url != "" {
		t.Doc = append(t.Doc, url)
	}
}

// helpers min function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
