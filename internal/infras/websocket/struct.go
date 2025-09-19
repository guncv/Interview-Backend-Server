package websocket

import "time"

type MsgConnectionEstablished struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

type MsgStartSessionConversation struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
}

type MsgUserAudioChunk struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	SegmentID string `json:"segment_id"`
}

type MsgInterviewerAudioChunk struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
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

type MsgUserFullTranscript struct {
	Type       string `json:"type"`
	Author     string `json:"author"`
	SessionID  string `json:"session_id"`
	SegmentID  string `json:"segment_id"`
	Transcript string `json:"transcript"`
}

type MsgInterviewerResp struct {
	Type             string `json:"type"`
	Author           string `json:"author"`
	SessionID        string `json:"session_id"`
	Message          string `json:"message"`
	StartedAt        string `json:"started_at,omitempty"`
	EndedAt          string `json:"ended_at,omitempty"`
	CurrentState     string `json:"current_state,omitempty"`
	AllowUserToSpeak bool   `json:"allow_user_to_speak,omitempty"`
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
	StartedAt  time.Time `json:"started_at,omitempty"`
	EndedAt    time.Time `json:"ended_at,omitempty"`
}

type msgError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
