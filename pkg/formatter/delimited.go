package formatter

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

type DelimitedFormatter struct {
	Comma       rune
	ShowPercent bool
}

func NewCSVFormatter() *DelimitedFormatter {
	return &DelimitedFormatter{Comma: ','}
}

func NewCSVFormatterWithPercent(showPercent bool) *DelimitedFormatter {
	return &DelimitedFormatter{Comma: ',', ShowPercent: showPercent}
}

func NewTSVFormatter() *DelimitedFormatter {
	return &DelimitedFormatter{Comma: '\t'}
}

func NewTSVFormatterWithPercent(showPercent bool) *DelimitedFormatter {
	return &DelimitedFormatter{Comma: '\t', ShowPercent: showPercent}
}

func (f *DelimitedFormatter) Format(w io.Writer, report *model.Report) error {
	writer := csv.NewWriter(w)
	writer.Comma = f.Comma

	// Header row
	header := []string{"path", "files", "added", "deleted", "net"}
	if f.ShowPercent {
		header = append(header, "percent")
	}
	if err := writer.Write(header); err != nil {
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
