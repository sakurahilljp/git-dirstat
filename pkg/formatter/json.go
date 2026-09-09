package formatter

import (
	"encoding/json"
	"io"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Format(w io.Writer, report *model.Report) error {
	rep := *report
	if rep.Entries == nil {
		rep.Entries = []model.Entry{}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rep)
}
