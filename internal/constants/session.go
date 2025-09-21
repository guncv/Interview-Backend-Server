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

	WebSocketMessageTypeHello                    = "hello"
	WebSocketMessageTypeStartSessionConversation = "start_session_conversation"
	WebSocketMessageTypeSegmentStart             = "segment_start"
	WebSocketMessageTypeSegmentEnd               = "segment_end"
	WebSocketBineryTypeAudioChunk                = "audio_chunk"
	WebSocketMessageTypeError                    = "error"
	WebSocketMessageTypeUserFullTranscript       = "user_full_transcript"
	WebSocketMessageTypeInterviewerResponse      = "interviewer_response"
	WebSocketMessageTypeConversationStarted      = "conversation_started"
	WebSocketMessageTypeInterviewTurnStart       = "interviewer_turn_start"
	WebSocketMessageTypeInterviewTurnEnd         = "interviewer_turn_end"

	// Additional message types for Python server compatibility
	WebSocketMessageTypeConnectionEstablished = "connection_established"
	WebSocketMessageTypeEcho                  = "echo"
	WebSocketMessageTypePing                  = "ping"
	WebSocketMessageTypePong                  = "pong"
	WebSocketMessageTypeClose                 = "close"

	WebSocketPingInterval                   = 2 * time.Second
	WebSocketPingDuration                   = 2 * time.Second
	WebSocketReadTimeout                    = 6 * time.Second
	WebSocketPongTimeout                    = 6 * time.Second
	WebSocketClientHandshakeTimeout         = 2 * time.Second
	WebSocketPreviousSegmentExpiredDuration = 10 * time.Second

	// Interviewer Constants
	ActorInterviewer = "interviewer"
	ActorUser        = "user"
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
