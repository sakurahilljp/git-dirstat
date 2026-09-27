package formatter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

type ChurnDelimitedFormatter struct {
	Comma       rune
	ShowPercent bool
}

func NewChurnCSVFormatterWithPercent(showPercent bool) *ChurnDelimitedFormatter {
	return &ChurnDelimitedFormatter{Comma: ',', ShowPercent: showPercent}
}

func NewChurnTSVFormatterWithPercent(showPercent bool) *ChurnDelimitedFormatter {
	return &ChurnDelimitedFormatter{Comma: '\t', ShowPercent: showPercent}
}

func (f *ChurnDelimitedFormatter) FormatChurn(w io.Writer, report *model.ChurnReport) error {
	writer := csv.NewWriter(w)
	writer.Comma = f.Comma

	header := []string{"path", "commits", "files", "added", "deleted", "churn"}
	if f.ShowPercent {
		header = append(header, "percent")
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, e := range report.Entries {
		row := []string{
			FormatChurnEntryPath(e, report.Target),
			strconv.Itoa(e.Commits),
			strconv.Itoa(e.Files),
			strconv.Itoa(e.Added),
			strconv.Itoa(e.Deleted),
			strconv.Itoa(e.Churn),
		}
		if f.ShowPercent {
			row = append(row, fmt.Sprintf("%.1f", e.Percent))
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
