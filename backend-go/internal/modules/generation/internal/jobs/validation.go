package generation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type markdownLine struct {
	text       string
	structural bool
}

func markedLines(markdown string) ([]markdownLine, bool) {
	out := []markdownLine{}
	fence := ""
	math := false
	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		structural := fence == "" && !math
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			marker := line[:3]
			if fence == "" {
				fence = marker
			} else if marker == fence {
				fence = ""
			}
			structural = false
		} else if fence == "" && strings.HasPrefix(line, "$$") {
			if strings.Count(line, "$$") == 1 {
				math = !math
			}
			structural = false
		}
		out = append(out, markdownLine{text: line, structural: structural})
	}
	return out, fence == "" && !math
}

// Only headings outside complete code and display-math fences are structural.
func structuralLines(markdown string) []string {
	out := []string{}
	fence := ""
	math := false
	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
			marker := line[:3]
			if fence == "" {
				fence = marker
			} else if marker == fence {
				fence = ""
			}
			continue
		}
		if fence != "" {
			continue
		}
		if strings.HasPrefix(line, "$$") {
			if strings.Count(line, "$$") == 1 {
				math = !math
			}
			continue
		}
		if !math {
			out = append(out, line)
		}
	}
	return out
}

var questionHeading = regexp.MustCompile(`^## 第([0-9]+)题$`)
var metadata = regexp.MustCompile(`^题型：\s*(single-choice|multiple-choice|true-false|short-answer|essay|code)\s*·\s*难度：\s*([1-5])/5$`)
var optionLine = regexp.MustCompile(`^([A-Z])[.、．)]\s*.+`)
var optionAnswer = regexp.MustCompile(`\b[A-Z]\b`)

// New practice rounds are objective only: the first answer line is the
// machine-gradable key the learner UI scores against.
var objectiveKinds = map[string]bool{"single-choice": true, "multiple-choice": true, "true-false": true}
var answerKey = regexp.MustCompile(`^答案[：:]\s*(.+?)\s*$`)
var choiceKey = regexp.MustCompile(`^[A-Z](?:\s*[、,，]\s*[A-Z])*$`)

func ValidatePractice(body string, count int) error {
	type item struct{ lines []markdownLine }
	items := []item{}
	lines, closed := markedLines(body)
	if !closed {
		return fmt.Errorf("unclosed output fence")
	}
	for _, line := range lines {
		if match := questionHeading.FindStringSubmatch(line.text); line.structural && match != nil {
			n, _ := strconv.Atoi(match[1])
			if n != len(items)+1 {
				return fmt.Errorf("question numbering")
			}
			items = append(items, item{})
			continue
		}
		if len(items) > 0 {
			items[len(items)-1].lines = append(items[len(items)-1].lines, line)
		}
	}
	if len(items) != count {
		return fmt.Errorf("question count")
	}
	lastDifficulty := 0
	difficulties := map[int]bool{}
	for _, it := range items {
		kind := ""
		difficulty := 0
		section := "question"
		question, answer, explanation, firstAnswerLine := "", "", "", ""
		options := map[string]bool{}
		for _, line := range it.lines {
			if !line.structural && (strings.HasPrefix(line.text, "```") || strings.HasPrefix(line.text, "~~~") || line.text == "$$") {
				continue
			}
			if m := metadata.FindStringSubmatch(line.text); line.structural && m != nil {
				if kind != "" {
					return fmt.Errorf("duplicate metadata")
				}
				kind = m[1]
				difficulty, _ = strconv.Atoi(m[2])
				continue
			}
			if line.structural && line.text == "### 参考答案" {
				section = "answer"
				continue
			}
			if line.structural && line.text == "### 解析" {
				section = "explanation"
				continue
			}
			switch section {
			case "question":
				question += line.text
				if m := optionLine.FindStringSubmatch(line.text); line.structural && m != nil {
					options[m[1]] = true
				}
			case "answer":
				if answer == "" && strings.TrimSpace(line.text) != "" {
					firstAnswerLine = strings.TrimSpace(line.text)
				}
				answer += line.text
			case "explanation":
				explanation += line.text
			}
		}
		if kind == "" || difficulty < lastDifficulty || question == "" || answer == "" || explanation == "" {
			return fmt.Errorf("incomplete question")
		}
		lastDifficulty = difficulty
		difficulties[difficulty] = true
		if !objectiveKinds[kind] {
			return fmt.Errorf("subjective question")
		}
		key := answerKey.FindStringSubmatch(firstAnswerLine)
		if key == nil {
			return fmt.Errorf("answer key line")
		}
		if kind == "true-false" && key[1] != "正确" && key[1] != "错误" {
			return fmt.Errorf("true-false answer")
		}
		if kind == "single-choice" || kind == "multiple-choice" {
			if !choiceKey.MatchString(key[1]) {
				return fmt.Errorf("choice answer")
			}
			selected := []string{}
			seen := map[string]bool{}
			for _, label := range optionAnswer.FindAllString(key[1], -1) {
				if !seen[label] {
					seen[label] = true
					selected = append(selected, label)
				}
			}
			if len(options) < 2 || len(selected) == 0 {
				return fmt.Errorf("choice answer")
			}
			for _, label := range selected {
				if !options[label] {
					return fmt.Errorf("nonexistent option")
				}
			}
			if kind == "single-choice" && len(selected) != 1 {
				return fmt.Errorf("single choice answer")
			}
			if kind == "multiple-choice" && len(selected) < 2 {
				return fmt.Errorf("multiple choice answer")
			}
		}
	}
	if count >= 5 && len(difficulties) != 5 {
		return fmt.Errorf("difficulty coverage")
	}
	return nil
}
