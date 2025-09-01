package constants

import (
	"time"
)

// Interview Session Constants
const (
	StatusPending   = "pending"
	StatusOnGoing   = "on_going"
	StatusCompleted = "completed"
	StatusAborted   = "aborted"
	StatusCancelled = "cancelled"
	StatusTimedOut  = "timed_out"

	ModalityVoiceChat = "voice_chat"

	WebSocketMessageTypeHello        = "hello"
	WebSocketMessageTypeSegmentStart = "segment_start"
	WebSocketMessageTypeSegmentEnd   = "segment_end"
	WebSocketBineryTypeAudioChunk    = "audio_chunk"
	WebSocketMessageTypeError        = "error"

	// Additional message types for Python server compatibility
	WebSocketMessageTypeConnectionEstablished = "connection_established"
	WebSocketMessageTypeEcho                  = "echo"
	WebSocketMessageTypePing                  = "ping"
	WebSocketMessageTypePong                  = "pong"
	WebSocketMessageTypeClose                 = "close"

	WebSocketPingInterval           = 2 * time.Second
	WebSocketPingDuration           = 2 * time.Second
	WebSocketReadTimeout            = 6 * time.Second
	WebSocketPongTimeout            = 4 * time.Second
	WebSocketClientHandshakeTimeout = 2 * time.Second
)

const (
	LanguageThai    = "thai"
	LanguageEnglish = "english"

	LanguageCodeThai    = "th-TH"
	LanguageCodeEnglish = "en-US"
)

var LanguageMapping = map[string]string{
	LanguageThai:    LanguageCodeThai,
	LanguageEnglish: LanguageCodeEnglish,
}
