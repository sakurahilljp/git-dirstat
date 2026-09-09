package formatter

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

type DelimitedFormatter struct {
	Comma rune
}

func NewCSVFormatter() *DelimitedFormatter {
	return &DelimitedFormatter{Comma: ','}
}

func NewTSVFormatter() *DelimitedFormatter {
	return &DelimitedFormatter{Comma: '\t'}
}

func (f *DelimitedFormatter) Format(w io.Writer, report *model.Report) error {
	writer := csv.NewWriter(w)
	writer.Comma = f.Comma

	// Header row
	if err := writer.Write([]string{"path", "files", "added", "deleted", "net"}); err != nil {
		return err
	}

	for _, e := range report.Entries {
		row := []string{
			FormatEntryPath(e, report.Target),
			strconv.Itoa(e.Files),
			strconv.Itoa(e.Added),
			strconv.Itoa(e.Deleted),
			strconv.Itoa(e.Net),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
