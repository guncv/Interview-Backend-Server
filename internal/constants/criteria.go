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

var StateToCriteriaStateMap = map[string]string{
	InterviewStateGreeting:           CriteriaInterviewStateGreetingName,
	InterviewStateIntro:              CriteriaInterviewStateIntroName,
	InterviewStateExperience:         CriteriaInterviewStateExperienceName,
	InterviewStateProject:            CriteriaInterviewStateProjectName,
	InterviewStateTechnicalQuestion:  CriteriaInterviewStateTechnicalQuestionName,
	InterviewStateBehavioralQuestion: CriteriaInterviewStateBehavioralQuestionName,
	InterviewStateUnknown:            CriteriaInterviewStateUnknown,
}

func GetCriteriaStateFromState(step string) string {
	if criteriaState, exists := StateToCriteriaStateMap[step]; exists {
		return criteriaState
	}

	return CriteriaInterviewStateUnknown
}
