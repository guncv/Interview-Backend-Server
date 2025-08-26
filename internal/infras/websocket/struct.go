package websocket

import "time"

type msgBase struct {
	V    int    `json:"v"`
	Type string `json:"type"`
}

type msgHello struct {
	msgBase
	SessionID string `json:"session_id"`
}

type msgSegmentStart struct {
	msgBase
	SegmentID  string `json:"segment_id"`
	SampleRate int    `json:"sample_rate"`
	Encoding   string `json:"encoding"`
	Channels   int    `json:"channels"`
	StartedAt  int64  `json:"started_at_ms,omitempty"`
}

type msgAudioChunk struct {
	V          int    `json:"v"`
	Type       string `json:"type"`
	SegmentID  string `json:"segment_id"`
	ChunkIndex int    `json:"index"`
	Timestamp  int64  `json:"timestamp"`
}

type msgSegmentEnd struct {
	msgBase
	SegmentID string `json:"segment_id"`
	EndedAt   int64  `json:"ended_at_ms,omitempty"`
}

type ConversationTurn struct {
	SessionID  string    `json:"session_id"`
	SegmentID  string    `json:"segment_id"`
	TurnType   string    `json:"turn_type"` // "user_audio", "asr_result", "tts_start", "tts_end"
	Content    string    `json:"content,omitempty"`
	TTSID      string    `json:"tts_id,omitempty"`
	Encoding   string    `json:"encoding,omitempty"`
	SampleRate int       `json:"sample_rate,omitempty"`
	Channels   int       `json:"channels,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	EndedAt    time.Time `json:"ended_at,omitempty"`
}

type msgASR struct {
	msgBase
	SegmentID string  `json:"segment_id"`
	Text      string  `json:"text"`
	IsFinal   bool    `json:"is_final"`
	Seq       int     `json:"seq,omitempty"`
	Stability float64 `json:"stability,omitempty"`
}

type msgEvaluation struct {
	msgBase
	SegmentID string  `json:"segment_id"`
	Score     float64 `json:"score"`
	Comment   string  `json:"comment"`
}

type msgTTSStart struct {
	msgBase
	SegmentID string `json:"segment_id"`
	TTSID     string `json:"tts_id"`
	Encoding  string `json:"encoding"` // "OPUS_OGG" | "OPUS_WEBM" | "PCM16"
}

type msgTTSEnd struct {
	msgBase
	SegmentID string `json:"segment_id"`
	TTSID     string `json:"tts_id"`
}

type msgDBAck struct {
	msgBase
	TurnID string   `json:"turn_id"`
	Saved  []string `json:"saved"`
}

type msgError struct {
	msgBase
	Code    string `json:"code"`
	Message string `json:"message"`
}
