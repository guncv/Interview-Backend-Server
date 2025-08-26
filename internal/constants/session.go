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
	WebSocketBineryTypeAudioChunk    = "audio_chunk"

	WebSocketMessageTypeASR        = "asr"
	WebSocketMessageTypeEvaluation = "evaluation"
	WebSocketMessageTypeTTSStart   = "tts_start"
	WebSocketMessageTypeTTSEnd     = "tts_end"
	WebSocketMessageTypeError      = "error"

	WebSocketPingInterval = 30 * time.Second
	WebSocketReadTimeout  = 60 * time.Second
)
