package generation

import (
	"context"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/modules/sources"
)

// Navigation is a hint, not a requirement: local OCR produces page text rather
// than a Markdown chapter tree. Seed the frozen readable body with citations;
// the model can follow continuations and organize chapters from that evidence.
func sourceMapContext(ctx context.Context, slug string, docs []sources.SourceDocument) (map[string][]sources.SourceOutlineNode, []sources.SourceReadData, error) {
	st, err := sources.New(slug)
	if err != nil {
		return nil, nil, err
	}
	reader := sources.NewReader(st)
	out := map[string][]sources.SourceOutlineNode{}
	samples := []sources.SourceReadData{}
	if len(docs) == 0 {
		return out, samples, problem("source-selection-required", 400)
	}
	budget := 16000 / len(docs)
	if budget > 4000 {
		budget = 4000
	}
	for _, d := range docs {
		_, _, body, err := st.ReadRevision(d.SourceID, d.RevisionID, d.ContentHash)
		if err != nil {
			return nil, nil, err
		}
		lines := strings.Count(string(body), "\n") + 1
		if len(body) > 0 && body[len(body)-1] == '\n' {
			lines--
		}
		result := reader.Outline(ctx, docs, sources.OutlineSourceArgs{SourceID: d.SourceID, RevisionID: d.RevisionID, Target: &sources.SourceOutlineTarget{Kind: "lines", Start: 1, End: lines}, MaxTokens: 1000})
		data, ok := result.Data.(sources.SourceOutlineData)
		if result.Status == "error" || !ok || len(data.Nodes) == 0 {
			return nil, nil, problem("source-verification-unavailable", 409)
		}
		read := reader.Read(ctx, docs, sources.ReadSourceArgs{SourceID: d.SourceID, RevisionID: d.RevisionID, Locator: data.Nodes[0].Locator, MaxTokens: budget})
		sample, ok := read.Data.(sources.SourceReadData)
		if read.Status == "error" || !ok || strings.TrimSpace(sample.Body) == "" {
			return nil, nil, problem("source-verification-unavailable", 409)
		}
		samples = append(samples, sample)
		hints := reader.Outline(ctx, docs, sources.OutlineSourceArgs{SourceID: d.SourceID, RevisionID: d.RevisionID, Limit: 8, MaxTokens: 500})
		if data, ok := hints.Data.(sources.SourceOutlineData); ok && hints.Status != "error" {
			out[d.SourceID] = data.Nodes
		}
	}
	return out, samples, nil
}
