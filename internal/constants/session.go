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

	WebSocketMessageTypeHello                     = "hello"
	WebSocketMessageTypeStartSessionConversation  = "start_session_conversation"
	WebSocketMessageTypeSegmentStart              = "segment_start"
	WebSocketMessageTypeSegmentEnd                = "segment_end"
	WebSocketBineryTypeAudioChunk                 = "audio_chunk"
	WebSocketMessageTypeError                     = "error"
	WebSocketMessageTypeUserFullTranscript        = "user_full_transcript"
	WebSocketMessageTypeInterviewerResponse       = "interviewer_response"
	WebSocketMessageTypeConversationStarted       = "conversation_started"
	WebSocketMessageTypeConversationStarting      = "conversation_starting"
	WebSocketMessageTypeInterviewTurnStart        = "interviewer_turn_start"
	WebSocketMessageTypeInterviewTurnEnd          = "interviewer_turn_end"
	WebSocketMessageTypeEndInterviewSession       = "end_interview_session"
	WebSocketMessageTypeSummarizeInterviewSession = "summarize_interview_session"
	WebSocketMessageTypeInterviewSessionTimedOut  = "interview_session_timed_out"
	WebSocketMessageTypeInactivityWarning         = "inactivity_warning"
	WebSocketMessageTypeActivityTimerReset        = "activity_timer_reset"

	// Additional message types for Python server compatibility
	WebSocketMessageTypeConnectionEstablished = "connection_established"
	WebSocketMessageTypeEcho                  = "echo"
	WebSocketMessageTypePing                  = "ping"
	WebSocketMessageTypePong                  = "pong"
	WebSocketMessageTypeClose                 = "close"

	WebSocketPingInterval                   = 2 * time.Second
	WebSocketPingDuration                   = 2 * time.Second
	WebSocketReadTimeout                    = 2 * time.Minute
	WebSocketPongTimeout                    = 6 * time.Second
	WebSocketClientHandshakeTimeout         = 2 * time.Second
	WebSocketPreviousSegmentExpiredDuration = 10 * time.Second
	WebSocketInactivityWarningTimeout       = 3 * time.Minute
	WebSocketInactivityTimeoutDuration      = 6 * time.Minute

	// Interviewer Constants
	ActorInterviewer = "interviewer"
	ActorUser        = "user"

	TurnNoDefault = 0

	BlankOverallSummaryMd    = "The interview session ended without any responses from the candidate."
	TimeOutMessage           = "The interview session has timed out due to inactivity. Please refresh the page and start a new session to continue."
	InactivityWarningMessage = "You have been inactive for 3 minutes. The session will timeout in 3 more minutes if no activity is detected."
)
