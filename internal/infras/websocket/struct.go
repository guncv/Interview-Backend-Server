package websocket

import "time"

type msgAudioChunk struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	SegmentID string `json:"segment_id"`
}

type msgSegmentStart struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	SegmentID string `json:"segment_id"`
	StartedAt int64  `json:"started_at_ms,omitempty"`
}

type msgSegmentEnd struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	SegmentID string `json:"segment_id"`
}

type ConversationTurn struct {
	SessionID  string    `json:"session_id"`
	SegmentID  string    `json:"segment_id"`
	TurnType   string    `json:"turn_type"`
	Content    string    `json:"content,omitempty"`
	TTSID      string    `json:"tts_id,omitempty"`
	Encoding   string    `json:"encoding,omitempty"`
	SampleRate int       `json:"sample_rate,omitempty"`
	Channels   int       `json:"channels,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at,omitempty"`
}

type msgASR struct {
	Type      string  `json:"type"`
	SegmentID string  `json:"segment_id"`
	Text      string  `json:"text"`
	IsFinal   bool    `json:"is_final"`
	Seq       int     `json:"seq,omitempty"`
	Stability float64 `json:"stability,omitempty"`
}

type msgEvaluation struct {
	Type      string  `json:"type"`
	SegmentID string  `json:"segment_id"`
	Score     float64 `json:"score"`
	Comment   string  `json:"comment"`
}

type msgTTSStart struct {
	Type      string `json:"type"`
	SegmentID string `json:"segment_id"`
	TTSID     string `json:"tts_id"`
	Encoding  string `json:"encoding"`
}

type msgTTSEnd struct {
	Type      string `json:"type"`
	SegmentID string `json:"segment_id"`
	TTSID     string `json:"tts_id"`
}

type msgDBAck struct {
	Type   string   `json:"type"`
	TurnID string   `json:"turn_id"`
	Saved  []string `json:"saved"`
}

type msgError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
