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
	InterviewStateGreetingColor           = "#4CAF50"
	InterviewStateIntroColor              = "#2196F3"
	InterviewStateExperienceColor         = "#FF9800"
	InterviewStateProjectColor            = "#9C27B0"
	InterviewStateTechnicalQuestionColor  = "#F44336"
	InterviewStateBehavioralQuestionColor = "#00BCD4"
	InterviewStateWrapUpColor             = "#795548"
	InterviewStateUnknownColor            = "#607D8B"
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

var StateToColorMap = map[string]string{
	InterviewStateGreeting:           InterviewStateGreetingColor,
	InterviewStateIntro:              InterviewStateIntroColor,
	InterviewStateExperience:         InterviewStateExperienceColor,
	InterviewStateProject:            InterviewStateProjectColor,
	InterviewStateTechnicalQuestion:  InterviewStateTechnicalQuestionColor,
	InterviewStateBehavioralQuestion: InterviewStateBehavioralQuestionColor,
	InterviewStateWrapUp:             InterviewStateWrapUpColor,
	InterviewStateUnknown:            InterviewStateUnknownColor,
}

func GetCriteriaStateFromState(step string) string {
	if criteriaState, exists := StateToCriteriaStateMap[step]; exists {
		return criteriaState
	}

	return CriteriaInterviewStateUnknown
}

func GetColorFromState(step string) string {
	if color, exists := StateToColorMap[step]; exists {
		return color
	}

	return InterviewStateUnknownColor
}
