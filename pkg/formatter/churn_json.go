package formatter

import (
	"encoding/json"
	"io"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

type ChurnJSONFormatter struct{}

func NewChurnJSONFormatter() *ChurnJSONFormatter {
	return &ChurnJSONFormatter{}
}

func (f *ChurnJSONFormatter) FormatChurn(w io.Writer, report *model.ChurnReport) error {
	rep := *report
	if rep.Entries == nil {
		rep.Entries = []model.ChurnEntry{}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rep)
}
