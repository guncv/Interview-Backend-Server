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
