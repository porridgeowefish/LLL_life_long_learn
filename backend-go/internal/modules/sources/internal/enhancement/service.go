package enhancement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xmz14/lll/backend-go/internal/idgen"
	projects "github.com/xmz14/lll/backend-go/internal/modules/projects"
	"github.com/xmz14/lll/backend-go/internal/modules/sources/internal/ocr"
	"github.com/xmz14/lll/backend-go/internal/modules/sources/internal/parser"
	"github.com/xmz14/lll/backend-go/internal/modules/sources/internal/render"
	"github.com/xmz14/lll/backend-go/internal/modules/sources/internal/store"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type task struct {
	slug, sourceID string
	st             *store.Store
	job            store.StructureJob
}

var manager = struct {
	sync.Mutex
	once    sync.Once
	queue   chan task
	running map[string]context.CancelFunc
	event   func(string, store.StructureJob)
}{queue: make(chan task, 16), running: map[string]context.CancelFunc{}}
var operation = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
var automaticQueue = make(chan task, 16)
var automaticOnce sync.Once

func NotifyAutomatic(slug string, st *store.Store, sourceID string) {
	automaticOnce.Do(func() {
		go func() {
			for t := range automaticQueue {
				Automatic(t.slug, t.st, t.sourceID)
			}
		}()
	})
	select {
	case automaticQueue <- task{slug: slug, st: st, sourceID: sourceID}:
	default:
	}
}
func ConfigureEvents(event func(string, store.StructureJob)) {
	manager.Lock()
	manager.event = event
	manager.Unlock()
	manager.once.Do(func() { go loop() })
	go recoverJobs()
}
func emit(slug string, j store.StructureJob) {
	manager.Lock()
	f := manager.event
	manager.Unlock()
	if f != nil {
		f(slug, j)
	}
}
func Create(ctx context.Context, slug, sourceID string, req store.StructureRequest, automatic bool) (store.StructureJob, error) {
	if ctx.Err() != nil {
		return store.StructureJob{}, ctx.Err()
	}
	if !operation.MatchString(req.OperationID) || req.RevisionID == "" || req.BaseDerivedVersionID == "" {
		return store.StructureJob{}, errors.New("invalid-arguments")
	}
	sort.Ints(req.Pages)
	pages := []int{}
	for _, n := range req.Pages {
		if n < 1 {
			return store.StructureJob{}, errors.New("invalid-pages")
		}
		if len(pages) == 0 || pages[len(pages)-1] != n {
			pages = append(pages, n)
		}
	}
	req.Pages = pages
	if len(pages) > 8 {
		return store.StructureJob{}, errors.New("invalid-pages")
	}
	st, e := store.New(slug)
	if e != nil {
		return store.StructureJob{}, e
	}
	manager.Lock()
	defer manager.Unlock()
	jobs, e := st.StructureJobs(sourceID)
	if e != nil {
		return store.StructureJob{}, e
	}
	intent, _ := json.Marshal(req)
	for _, j := range jobs {
		if j.OperationID == req.OperationID {
			old, _ := json.Marshal(j.Intent)
			if string(old) != string(intent) {
				return j, errors.New("operation-conflict")
			}
			return j, nil
		}
	}
	for _, j := range jobs {
		if j.Status == "queued" || j.Status == "running" {
			return j, errors.New("structure-active")
		}
	}
	if len(manager.queue) >= cap(manager.queue) {
		return store.StructureJob{}, errors.New("structure-queue-full")
	}
	source, rev, path, e := st.OriginalPath(sourceID)
	if e != nil {
		return store.StructureJob{}, e
	}
	if rev.RevisionID != req.RevisionID {
		return store.StructureJob{}, store.ErrRevisionChanged
	}
	if rev.DerivedVersionID != req.BaseDerivedVersionID {
		return store.StructureJob{}, store.ErrDerivedConflict
	}
	if !strings.HasSuffix(strings.ToLower(rev.Original.Filename), ".pdf") {
		return store.StructureJob{}, errors.New("unsupported-structure-format")
	}
	p, e := ocr.ResolvePageProvider(req.ProviderID, req.Model)
	if e != nil {
		return store.StructureJob{}, e
	}
	var structure *parser.Structure
	if source.Status == "ready" {
		structure, e = st.ReadStructure(sourceID, rev.RevisionID, "")
	} else {
		var result parser.Result
		result, e = parser.Extract(ctx, path, rev.Original.Filename)
		structure = result.Structure
		if parser.ErrorCode(e) == "no_extractable_text" {
			e = nil
		}
	}
	if e != nil || structure == nil {
		return store.StructureJob{}, errors.New("structure-unavailable")
	}
	valid := map[int]bool{}
	recognized := map[int]bool{}
	for _, u := range structure.Units {
		if u.Kind == "page" {
			valid[u.Number] = true
			for _, b := range u.Blocks {
				if b.Provenance == "vision" {
					recognized[u.Number] = true
				}
			}
		}
	}
	for _, n := range structure.Coverage.UnreadUnits {
		valid[n] = true
	}
	if len(pages) == 0 {
		for n := range valid {
			if !recognized[n] {
				pages = append(pages, n)
			}
		}
		sort.Ints(pages)
		pages = pages[:min(8, len(pages))]
	}
	if len(pages) == 0 {
		return store.StructureJob{}, errors.New("no-candidate-pages")
	}
	for _, n := range pages {
		if !valid[n] {
			return store.StructureJob{}, errors.New("invalid-pages")
		}
	}
	now := time.Now().UTC()
	j := store.StructureJob{ID: idgen.New("structure"), OperationID: req.OperationID, Status: "queued", RevisionID: rev.RevisionID, BaseDerivedVersionID: rev.DerivedVersionID, ProviderID: p.ID, Model: p.Model, Pages: pages, Automatic: automatic, OriginalHash: rev.Original.SHA256, Intent: req, CreatedAt: now, UpdatedAt: now}
	j, e = st.SaveStructureJob(sourceID, j)
	if e != nil {
		return j, e
	}
	manager.once.Do(func() { go loop() })
	manager.queue <- task{slug, sourceID, st, j}
	return j, nil
}
func Automatic(slug string, st *store.Store, sourceID string) {
	if _, e := ocr.ResolvePageProvider("", ""); e != nil {
		return
	}
	_, rev, e := st.Get(sourceID)
	if e != nil || !strings.HasSuffix(strings.ToLower(rev.Original.Filename), ".pdf") {
		return
	}
	ctx, c := context.WithTimeout(context.Background(), 45*time.Second)
	defer c()
	_, _ = Create(ctx, slug, sourceID, store.StructureRequest{OperationID: "automatic_" + rev.RevisionID, RevisionID: rev.RevisionID, BaseDerivedVersionID: rev.DerivedVersionID}, true)
}
func Get(slug, sourceID, jobID string) (store.StructureJob, error) {
	st, e := store.New(slug)
	if e != nil {
		return store.StructureJob{}, e
	}
	return st.GetStructureJob(sourceID, jobID)
}
func Cancel(slug, sourceID, jobID string) (store.StructureJob, error) {
	st, e := store.New(slug)
	if e != nil {
		return store.StructureJob{}, e
	}
	j, e := st.CancelStructureJob(sourceID, jobID)
	if e == nil {
		manager.Lock()
		cancel := manager.running[jobID]
		manager.Unlock()
		if cancel != nil {
			cancel()
		}
		emit(slug, j)
	}
	return j, e
}
func loop() {
	for t := range manager.queue {
		run(t)
	}
}
func run(t task) {
	j, e := t.st.GetStructureJob(t.sourceID, t.job.ID)
	if e != nil || j.Status != "queued" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	manager.Lock()
	manager.running[j.ID] = cancel
	manager.Unlock()
	defer func() { manager.Lock(); delete(manager.running, j.ID); manager.Unlock() }()
	j.Status = "running"
	j, e = t.st.SaveStructureJob(t.sourceID, j)
	if e != nil || j.Status != "running" {
		return
	}
	emit(t.slug, j)
	content, structure, images, e := execute(ctx, t, &j)
	if e == nil && ctx.Err() == nil {
		j, e = t.st.CommitStructureJob(t.sourceID, j.ID, content, structure, images)
	}
	if e != nil || ctx.Err() != nil {
		latest, readErr := t.st.GetStructureJob(t.sourceID, j.ID)
		if readErr != nil {
			return
		}
		j = latest
		if j.Status == "cancelled" || j.Status == "succeeded" {
			emit(t.slug, j)
			return
		}
		j.Status = "failed"
		j.ErrorCode = "structure-failed"
		if ctx.Err() != nil {
			j.Status = "interrupted"
			j.ErrorCode = "structure-interrupted"
		}
		if errors.Is(e, store.ErrDerivedConflict) {
			j.ErrorCode = "derived-conflict"
		}
		if e != nil && (e.Error() == "automatic-budget-limit" || e.Error() == "vision-output-truncated" || e.Error() == "vision-output-limit") {
			j.ErrorCode = e.Error()
		}
		j, _ = t.st.SaveStructureJob(t.sourceID, j)
	}
	emit(t.slug, j)
}

type usage struct {
	RequestID  string         `json:"requestId"`
	JobID      string         `json:"jobId"`
	Attempt    int            `json:"attempt"`
	ProviderID string         `json:"providerId"`
	Model      string         `json:"model"`
	State      string         `json:"state"`
	Tokens     map[string]int `json:"tokens,omitempty"`
	Known      bool           `json:"known"`
	CreatedAt  time.Time      `json:"createdAt"`
	Kind       string         `json:"kind"`
}

func execute(ctx context.Context, t task, j *store.StructureJob) (string, *parser.Structure, map[string][]byte, error) {
	src, rev, path, e := t.st.OriginalPath(t.sourceID)
	if e != nil {
		return "", nil, nil, e
	}
	if rev.RevisionID != j.RevisionID || rev.Original.SHA256 != j.OriginalHash {
		return "", nil, nil, store.ErrRevisionChanged
	}
	if rev.DerivedVersionID != j.BaseDerivedVersionID {
		return "", nil, nil, store.ErrDerivedConflict
	}
	provider, e := ocr.ResolvePageProvider(j.ProviderID, j.Model)
	if e != nil {
		return "", nil, nil, e
	}
	document, e := os.ReadFile(path)
	if e != nil {
		return "", nil, nil, e
	}
	if len(document) > 100<<20 {
		return "", nil, nil, errors.New("resource-limit")
	}
	body := ""
	var baseline *parser.Structure
	if src.Status == "ready" {
		body, e = t.st.ReadContent(t.sourceID)
		if e != nil {
			return "", nil, nil, e
		}
		baseline, e = t.st.ReadStructure(t.sourceID, j.RevisionID, "")
	} else {
		result, err := parser.Extract(ctx, path, rev.Original.Filename)
		if err != nil && parser.ErrorCode(err) != "no_extractable_text" {
			return "", nil, nil, err
		}
		body = result.Content
		baseline = result.Structure
	}
	if baseline == nil {
		return "", nil, nil, errors.New("structure-unavailable")
	}
	replacements := map[int]string{}
	images := map[string][]byte{}
	renderDuration := time.Duration(0)
	for _, page := range j.Pages {
		if ctx.Err() != nil {
			return "", nil, nil, ctx.Err()
		}
		start := time.Now()
		remaining := 60*time.Second - renderDuration
		if remaining <= 0 {
			return "", nil, nil, errors.New("render-batch-limit")
		}
		renderCtx, stop := context.WithTimeout(ctx, min(15*time.Second, remaining))
		image, e := render.Render(renderCtx, document, page)
		stop()
		renderDuration += time.Since(start)
		if e != nil {
			return "", nil, nil, e
		}
		if e = t.st.ReserveStructureRequest(t.sourceID, j, page); e != nil {
			return "", nil, nil, e
		}
		requestID := j.Requests[len(j.Requests)-1]
		record := usage{RequestID: requestID, JobID: j.ID, Attempt: 1, ProviderID: j.ProviderID, Model: j.Model, State: "request-started", CreatedAt: time.Now().UTC(), Kind: "document-structure"}
		if e = t.st.AppendStructureUsage(t.sourceID, j.ID, record); e != nil {
			return "", nil, nil, e
		}
		result, callErr := ocr.ExtractPage(ctx, provider, image)
		record.State = "request-finished"
		record.Tokens = result.Tokens
		record.Known = result.Known
		record.CreatedAt = time.Now().UTC()
		if e = t.st.AppendStructureUsage(t.sourceID, j.ID, record); e != nil {
			return "", nil, nil, e
		}
		if callErr != nil {
			return "", nil, nil, callErr
		}
		replacements[page] = fmt.Sprintf("## 第 %d 页\n\n%s\n\n", page, result.Content)
		images[fmt.Sprintf("images/page-%d.png", page)] = image
	}
	content, structure := merge(body, baseline, replacements)
	if len(content) > 4<<20 {
		return "", nil, nil, errors.New("vision-output-limit")
	}
	return content, structure, images, nil
}

func CancelProject(slug string) error {
	st, e := store.New(slug)
	if e != nil {
		return e
	}
	list, e := st.List()
	if e != nil {
		return e
	}
	for _, src := range list {
		if src.Status == "deleted" || src.Status == "tombstoned" {
			continue
		}
		jobs, e := st.StructureJobs(src.SourceID)
		if e != nil {
			return e
		}
		for _, j := range jobs {
			if j.Status == "queued" || j.Status == "running" {
				if _, e = Cancel(slug, src.SourceID, j.ID); e != nil {
					return e
				}
			}
		}
	}
	return nil
}

func CancelSource(slug, sourceID string) error {
	st, e := store.New(slug)
	if e != nil {
		return e
	}
	jobs, e := st.StructureJobs(sourceID)
	if e != nil {
		return e
	}
	for _, j := range jobs {
		if j.Status == "queued" || j.Status == "running" {
			if _, e = Cancel(slug, sourceID, j.ID); e != nil {
				return e
			}
		}
	}
	return nil
}
func merge(body string, baseline *parser.Structure, replacements map[int]string) (string, *parser.Structure) {
	var out strings.Builder
	result := &parser.Structure{SchemaVersion: 1, ParserVersion: parser.ParserVersion}
	pages := append([]parser.Unit(nil), baseline.Units...)
	sort.SliceStable(pages, func(i, j int) bool { return pages[i].Number < pages[j].Number })
	recognized := map[int]bool{}
	for _, u := range pages {
		if u.Kind != "page" {
			continue
		}
		oldStart, oldEnd := 0, 0
		for i, b := range u.Blocks {
			if i == 0 {
				oldStart = b.Range.Start
			}
			oldEnd = max(oldEnd, b.Range.End)
		}
		text := replacements[u.Number]
		provenance := "local"
		if text != "" {
			provenance = "vision"
			recognized[u.Number] = true
		} else if oldEnd > oldStart && oldEnd <= len(body) {
			text = body[oldStart:oldEnd]
		}
		start := out.Len()
		out.WriteString(text)
		if provenance == "local" {
			u.Blocks = append([]parser.Block(nil), u.Blocks...)
			for i := range u.Blocks {
				b := &u.Blocks[i]
				b.Range.Start += start - oldStart
				b.Range.End += start - oldStart
				b.TableRows = append([]parser.ByteRange(nil), b.TableRows...)
				for j := range b.TableRows {
					b.TableRows[j].Start += start - oldStart
					b.TableRows[j].End += start - oldStart
				}
			}
		} else {
			local := parser.StructureFromMarkdown(text, provenance)
			u.Blocks = nil
			for _, lu := range local.Units {
				for _, b := range lu.Blocks {
					b.Range.Start += start
					b.Range.End += start
					for i := range b.TableRows {
						b.TableRows[i].Start += start
						b.TableRows[i].End += start
					}
					b.ID = fmt.Sprintf("page_%d_%s", u.Number, b.ID)
					u.Blocks = append(u.Blocks, b)
				}
			}
		}
		result.Units = append(result.Units, u)
		if text != "" {
			result.Coverage.ExtractedUnits = append(result.Coverage.ExtractedUnits, u.Number)
		} else {
			result.Coverage.UnreadUnits = append(result.Coverage.UnreadUnits, u.Number)
		}
	}
	for _, n := range baseline.Coverage.UnreadUnits {
		found := false
		for _, u := range result.Units {
			if u.Number == n {
				found = true
			}
		}
		if !found {
			if text := replacements[n]; text != "" {
				start := out.Len()
				out.WriteString(text)
				result.Units = append(result.Units, parser.Unit{ID: fmt.Sprintf("page_%d", n), Kind: "page", Number: n, Blocks: []parser.Block{{ID: fmt.Sprintf("page_%d_vision", n), Type: "text", Range: parser.ByteRange{Start: start, End: out.Len()}, Representation: "markdown", Provenance: "vision"}}})
				recognized[n] = true
			} else {
				result.Coverage.UnreadUnits = append(result.Coverage.UnreadUnits, n)
			}
		}
	}
	result.Coverage.CompleteWithinScope = len(result.Coverage.UnreadUnits) == 0 && baseline.Coverage.CompleteWithinScope
	result.Coverage.Reasons = append(result.Coverage.Reasons, "vision-is-not-verbatim")
	_ = recognized
	return out.String(), result
}
func recoverJobs() {
	units, e := projects.IndexAll()
	if e != nil {
		return
	}
	for _, unit := range units {
		if unit.ProjectType != projects.ProjectTypeSystemLearning && unit.ProjectType != projects.ProjectTypeDisciplineMap {
			continue
		}
		st, e := store.New(unit.Slug)
		if e != nil {
			continue
		}
		sources, e := st.List()
		if e != nil {
			continue
		}
		for _, src := range sources {
			jobs, e := st.StructureJobs(src.SourceID)
			if e != nil {
				continue
			}
			for _, j := range jobs {
				if j.Status == "running" {
					_, _ = st.RecoverStructureJob(src.SourceID, j.ID)
				} else if j.Status == "queued" {
					manager.Lock()
					select {
					case manager.queue <- task{unit.Slug, src.SourceID, st, j}:
					default:
					}
					manager.Unlock()
				}
			}
		}
	}
}
