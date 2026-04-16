package catalog

import (
	"encoding/csv"
	"os"
	"sort"
	"strings"
)

type Work struct {
	ID            string
	Title         string
	TitleReading  string
	Subtitle      string
	PublishedYear string
	NDC           string
	Kana          string // copyright flag
	TextURL       string
	AuthorID      string
	LastName      string
	FirstName     string
}

func (w *Work) AuthorName() string {
	return w.LastName + w.FirstName
}

// DisplayTitle returns "Title　Subtitle" when subtitle exists.
func (w *Work) DisplayTitle() string {
	if w.Subtitle == "" {
		return w.Title
	}
	return w.Title + "　" + w.Subtitle
}

type Author struct {
	ID          string
	LastName    string
	FirstName   string
	Reading     string // 姓読みソート用
	RomanLast   string
	RomanFirst  string
	Works       []*Work
}

func (a *Author) Name() string {
	return a.LastName + " " + a.FirstName
}

func Load(csvPath string) ([]*Author, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	authorMap := map[string]*Author{}

	for i, rec := range records {
		if i == 0 {
			continue // header
		}
		if len(rec) < 46 {
			continue
		}

		// copyright check: skip if work or author has copyright
		workCopyright := rec[10]
		if workCopyright == "あり" {
			continue
		}

		textURL := rec[45]
		if textURL == "" {
			continue
		}

		authorID := rec[14]
		work := &Work{
			ID:            rec[0],
			Title:         rec[1],
			TitleReading:  rec[2],
			Subtitle:      rec[4],
			NDC:           rec[8],
			Kana:          workCopyright,
			TextURL:       textURL,
			AuthorID:      authorID,
			LastName:      rec[15],
			FirstName:     rec[16],
		}

		// extract year from 公開日 (rec[11]: YYYY-MM-DD)
		if len(rec[11]) >= 4 {
			work.PublishedYear = rec[11][:4]
		}

		a, ok := authorMap[authorID]
		if !ok {
			a = &Author{
				ID:         authorID,
				LastName:   rec[15],
				FirstName:  rec[16],
				Reading:    rec[19],
				RomanLast:  rec[21],
				RomanFirst: rec[22],
			}
			authorMap[authorID] = a
		}
		a.Works = append(a.Works, work)
	}

	authors := make([]*Author, 0, len(authorMap))
	for _, a := range authorMap {
		authors = append(authors, a)
	}
	sort.Slice(authors, func(i, j int) bool {
		return authors[i].Reading < authors[j].Reading
	})

	return authors, nil
}

func SanitizeFilename(s string) string {
	r := strings.NewReplacer(
		"/", "／", "\\", "＼", ":", "：", "*", "＊",
		"?", "？", "\"", "＂", "<", "＜", ">", "＞", "|", "｜",
		" ", "_",
	)
	return r.Replace(s)
}

func FilterAuthors(authors []*Author, query string) []*Author {
	if query == "" {
		return authors
	}
	q := strings.ToLower(query)
	var result []*Author
	for _, a := range authors {
		name := strings.ToLower(a.LastName + a.FirstName + a.Reading + a.RomanLast + a.RomanFirst)
		if strings.Contains(name, q) {
			result = append(result, a)
		}
	}
	return result
}
