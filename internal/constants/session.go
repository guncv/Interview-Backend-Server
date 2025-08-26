package constants

import "time"

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
	WebSocketMessageTypeStopTTS      = "stop_tts"
	WebSocketMessageTypeError        = "error"

	WebSocketPingInterval = 30 * time.Second
	WebSocketReadTimeout  = 60 * time.Second
)
