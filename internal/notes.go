package calendar

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var Topics = []string{"programming", "system_design", "algorithms", "linux", "general"}

var TopicLabels = map[string]string{
	"programming":   "Programming",
	"system_design": "System Design",
	"algorithms":    "Algo & DS",
	"linux":         "Linux",
	"general":       "General",
}

const notesDir = "notes"

type Note struct {
	Title    string
	Filename string
	Topic    string
}

func notesDirForTopic(topic string) string {
	return filepath.Join(notesDir, topic)
}

func ListNotes(topic string) ([]Note, error) {
	dir := notesDirForTopic(topic)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []Note{}, nil
	}
	if err != nil {
		return nil, err
	}
	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		notes = append(notes, Note{
			Title:    filenameToTitle(e.Name()),
			Filename: e.Name(),
			Topic:    topic,
		})
	}
	return notes, nil
}

func LoadNote(topic, filename string) (string, error) {
	path := filepath.Join(notesDirForTopic(topic), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func SaveNote(topic, filename, content string) error {
	dir := notesDirForTopic(topic)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

func DeleteNote(topic, filename string) error {
	return os.Remove(filepath.Join(notesDirForTopic(topic), filename))
}

func CreateNote(topic, title, content string) (Note, error) {
	filename := TitleToFilename(title)
	if err := SaveNote(topic, filename, content); err != nil {
		return Note{}, err
	}
	return Note{Title: title, Filename: filename, Topic: topic}, nil
}

func TitleToFilename(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' {
			b.WriteByte('_')
		}
	}
	name := b.String()
	re := regexp.MustCompile(`_+`)
	name = re.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		name = "note"
	}
	return name + ".md"
}

func filenameToTitle(filename string) string {
	name := strings.TrimSuffix(filename, ".md")
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}
