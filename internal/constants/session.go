package constants

import (
	"time"
)

const (
	StatusPending         = "pending"
	StatusOnGoing         = "on_going"
	StatusCompleted       = "completed"
	StatusAborted         = "aborted"
	StatusCancelled       = "cancelled"
	StatusTimedOut        = "timed_out"
	StatusAlreadyTimedOut = "already_timed_out"

	ModalityVoiceChat = "voice_chat"

	WebSocketMessageTypeHello                            = "hello"
	WebSocketMessageTypeStartSessionConversation         = "start_session_conversation"
	WebSocketMessageTypeSegmentStart                     = "segment_start"
	WebSocketMessageTypeSegmentEnd                       = "segment_end"
	WebSocketBineryTypeAudioChunk                        = "audio_chunk"
	WebSocketMessageTypeError                            = "error"
	WebSocketMessageTypeUserFullTranscript               = "user_full_transcript"
	WebSocketMessageTypeInterviewerResponse              = "interviewer_response"
	WebSocketMessageTypeConversationStarted              = "conversation_started"
	WebSocketMessageTypeConversationStarting             = "conversation_starting"
	WebSocketMessageTypeInterviewTurnStart               = "interviewer_turn_start"
	WebSocketMessageTypeInterviewTurnEnd                 = "interviewer_turn_end"
	WebSocketMessageTypeEndInterviewSession              = "end_interview_session"
	WebSocketMessageTypeSummarizeInterviewSession        = "summarize_interview_session"
	WebSocketMessageTypeInterviewSessionTimedOut         = "interview_session_timed_out"
	WebSocketMessageTypeInterviewSessionAlreadyTimedOut  = "interview_session_already_timed_out"
	WebSocketMessageTypeInterviewSessionAlreadyCompleted = "interview_session_already_completed"
	WebSocketMessageTypeInactivityWarning                = "inactivity_warning"
	WebSocketMessageTypeInterviewCompleted               = "interview_completed"
	WebSocketMessageTypeUserCompleteSession              = "user_complete_session"

	WebSocketMessageTypeConnectionEstablished = "connection_established"
	WebSocketMessageTypeEcho                  = "echo"
	WebSocketMessageTypePing                  = "ping"
	WebSocketMessageTypePong                  = "pong"
	WebSocketMessageTypeClose                 = "close"

	WebSocketPingInterval                   = 2 * time.Second
	WebSocketPingDuration                   = 2 * time.Second
	WebSocketPongTimeout                    = 6 * time.Second
	WebSocketClientHandshakeTimeout         = 2 * time.Second
	WebSocketPreviousSegmentExpiredDuration = 10 * time.Second
	WebSocketInactivityWarningTimeout       = 3 * time.Minute
	WebSocketInactivityTimeoutDuration      = 6 * time.Minute

	WebSocketInactivityMonitorInterval = 10 * time.Second

	// Interviewer Constants
	ActorInterviewer = "interviewer"
	ActorUser        = "user"

	TurnNoDefault = 0

	BlankOverallSummaryMd    = "The interview session ended without any responses from the candidate."
	TimeOutMessage           = "Your interview session ended because there was no activity for a while. Don’t worry — you can refresh the page to start a new session whenever you’re ready."
	InactivityWarningMessage = "It looks like you’ve been inactive for a few minutes. Please continue soon — the session will close in 3 minutes if no response is received."
)
