package entities

type GetRubricWithCriteriaByNameResp struct {
	RubricID            string
	RubricName          string
	RubricDescriptionMd string
	RubricVersionLabel  string
	Criteria            []CritetiaRow
}

type CritetiaRow struct {
	CriterionID            string `json:"criterion_id"`
	CriterionCode          string `json:"criterion_code"`
	CriterionName          string `json:"criterion_name"`
	CriterionDescriptionMd string `json:"criterion_description_md"`
	CriterionWeight        string `json:"criterion_weight"`
	CriterionMaxScore      string `json:"criterion_max_score"`
}

type ListAllRubricsAndCriteriaResp struct {
	Rubrics []RubricAndCriteriaRow `json:"rubrics"`
}

type RubricAndCriteriaRow struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	DescriptionMd string         `json:"description_md"`
	Criteria      []CriterionRow `json:"criteria"`
}

type CriterionRow struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DescriptionMd string `json:"description_md"`
	Percentage    string `json:"percentage"`
	Color         string `json:"color"`
}

type GetPhraseEvaluationsWithCriteriaResp struct {
	PhraseEvaluations []PhraseEvaluations `json:"phrase_evaluations"`
}

type PhraseEvaluations struct {
	StateID      string                     `json:"state_id"`
	StateName    string                     `json:"state_name"`
	OverallScore float64                    `json:"overall_score"`
	MaxScore     float64                    `json:"max_score"`
	OverallColor string                     `json:"overall_color"`
	Criteria     []PhraseEvaluationCriteria `json:"criteria"`
}

type PhraseEvaluationCriteria struct {
	CriteriaID      string  `json:"criteria_id"`
	CriteriaName    string  `json:"criteria_name"`
	CriteriaScore   float64 `json:"criteria_score"`
	MaxScore        float64 `json:"max_score"`
	CriteriaColor   string  `json:"criteria_color"`
	CriteriaComment string  `json:"criteria_comment"`
}
