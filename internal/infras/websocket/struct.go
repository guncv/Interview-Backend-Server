package websocket

import "time"

type MsgAudioChunk struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	SegmentID string `json:"segment_id"`
}

type MsgSegmentStart struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	SegmentID string `json:"segment_id"`
}

type MsgSegmentEnd struct {
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

type msgError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
