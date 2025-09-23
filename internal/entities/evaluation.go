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
	Weight        string `json:"weight"`
}
