# Iteration 16 Test Plan

- Source-store tests require exactly `content.md`; source-processing commit tests
  reject more than one derived file and preserve processing/ready transitions.
- Teacher context tests assert map outlines appear only in system context.
- Task-prompt and commit tests verify consolidate defaults to Intro+Body and
  writes Practice only for explicit question objectives.
- Usage-store and HTTP tests cover normalized provider usage, append/read,
  pagination, and absent usage.
- React tests cover processing source presentation, composer reference selection,
  citation chips, token-page pagination, and removed Agent navigation.
- Run `npm run check:full`; browser E2E covers upload → processing → citation →
  teacher turn → confirmed consolidation and is performed by the learner.
