package constants

const (
	CurrentCriteriaVersion = "v1.0"
)

const (
	InterviewStateGreeting           = "Greeting"
	InterviewStateIntro              = "Intro"
	InterviewStateExperience         = "Experience"
	InterviewStateProject            = "Project"
	InterviewStateTechnicalQuestion  = "Technical Question"
	InterviewStateBehavioralQuestion = "Behavioral Question"
	InterviewStateWrapUp             = "Wrap Up"
	InterviewStateUnknown            = "Unknown"
)

const (
	CriteriaInterviewStateGreetingName           = "Greeting Rubric"
	CriteriaInterviewStateIntroName              = "Intro Rubric"
	CriteriaInterviewStateExperienceName         = "Experience Rubric"
	CriteriaInterviewStateProjectName            = "Project Rubric"
	CriteriaInterviewStateTechnicalQuestionName  = "Technical Rubric"
	CriteriaInterviewStateBehavioralQuestionName = "Behavioral Rubric"
	CriteriaInterviewStateUnknown                = "General Rubric"
)

const (
	CriteriaInterviewStepGreeting   = "greeting"
	CriteriaInterviewStepIntro      = "intro"
	CriteriaInterviewStepExperience = "experience"
	CriteriaInterviewStepProject    = "project"
	CriteriaInterviewStepTechnical  = "technical"
	CriteriaInterviewStepBehavioral = "behavioral"
	CriteriaInterviewStepWrapUp     = "wrap_up"
	CriteriaInterviewStepUnknown    = "unknown"
)

var StateToCriteriaStateMap = map[string]string{
	InterviewStateGreeting:           CriteriaInterviewStepGreeting,
	InterviewStateIntro:              CriteriaInterviewStepIntro,
	InterviewStateExperience:         CriteriaInterviewStepExperience,
	InterviewStateProject:            CriteriaInterviewStepProject,
	InterviewStateTechnicalQuestion:  CriteriaInterviewStepTechnical,
	InterviewStateBehavioralQuestion: CriteriaInterviewStepBehavioral,
	InterviewStateWrapUp:             CriteriaInterviewStepWrapUp,
	InterviewStateUnknown:            CriteriaInterviewStepUnknown,
}

func GetCriteriaStateFromState(step string) string {
	if criteriaState, exists := StateToCriteriaStateMap[step]; exists {
		return criteriaState
	}

	return CriteriaInterviewStepUnknown
}
