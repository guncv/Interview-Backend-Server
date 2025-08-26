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

	WebSocketPingInterval = 30 * time.Second
	WebSocketPingDuration = 5 * time.Second
	WebSocketReadTimeout  = 60 * time.Second
	WebSocketPongTimeout  = 10 * time.Second
)
