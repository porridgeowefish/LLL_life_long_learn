# Iteration 07 Delivery Notes

Status: delivered
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: iteration 07 alignment, planning record, and delivery evidence.

## Product alignment

The product contract was confirmed on 2026-07-14:

```text
学科总览展示可能学习什么；左侧栏只展示已经创建了什么。
```

The learner chooses between a discipline map and a system-learning project.
AI provides optional advice but does not own the decision. A discipline map has
one overview and creates no branch folders. Only a learner-confirmed deep dive
becomes a real, ordinary five-zone project classified inside the map-backed folder.

## Implementation status

Delivered implementation:

```text
typed discipline-map and system-learning state with legacy defaulting
separate map and five-zone filesystem skeletons
single flat project-creation path with no parent relation
project list/detail type metadata, flat indexing, and map zone guards
stateless multi-turn AI project-type advice with no creation or persistence side effect
icon-led advice entry in the upper-right of the project-type selector
short discipline-map creation form and a scrolling system-learning form with a fixed footer
type-specific project briefs that keep ability, completion, and active-phase context out of discipline maps
registered encyclopedia agent and versioned outline/body charter
project-level prompt assembly with no synthetic sixth learning zone
explicit encyclopedia invocation through the normal Session -> run package -> visible native Agent CLI path
root overview artifact watching and query invalidation
Word-style table of contents derived from the agent-authored heading hierarchy
one inline deep-dive action beside every H3 key concept or branch
map overview page backed by the existing sidebar folder model
confirmed flat system-learning creation classified inside that map-backed folder
inline map-topic selection with editable title prefilling
Intro quick prerequisite bridge with no project creation
updated Intro charter and prerequisite primitive
```

## Test runs

Automated repair validation on 2026-07-14:

```text
Go targeted tests: promptassembly + server + artifactwatch passed
Frontend targeted tests: DisciplineOverview + ProjectTypeAdvisorDialog passed
Frontend production TypeScript/Vite build passed
Real-browser validation intentionally skipped at maintainer request
```

Type-specific project-brief repair validation on 2026-07-14:

```text
Go targeted tests: backend-go/internal/workspace passed
Frontend targeted tests: CreateProjectModal passed (3/3)
Frontend production TypeScript/Vite build passed
Real-browser validation intentionally skipped at maintainer request
```

The project-brief repair additionally verifies that map creation omits learning
fields at the client boundary and that the backend ignores them even when they
are submitted. Existing map briefs were normalized without changing their
generated overviews; learner-authored intent was retained as map scope notes.

The earlier direct-provider browser result is not accepted as evidence for the
encyclopedia Agent. It exposed the contract violation that this repair removes.
The still-valid automated and earlier interaction evidence is:

- `问问 AI` opened from an otherwise empty creation form, completed a temporary
  conversation, recommended `学科地图`, and applied the choice without creating
  or persisting a project.
- The configured provider returned malformed JSON with unescaped quotation
  marks during acceptance. The advice endpoint recovered the known fields and
  returned a normal user-facing answer instead of raw JSON or a silent failure.
- The creation footer stayed visible while the system-learning fields scrolled;
  choosing `学科地图` reduced the form to the fields that shape actually needs.
- `概率论与数理统计` rendered as a top-level, clickable sidebar folder overview,
  while its confirmed deep dives remained ordinary child rows managed by the
  existing folder classification model.
- The generation route now returns a normal Agent session only after requesting
  the shared native CLI launcher; tests assert the versioned charter and
  `overview.md` output contract are present in the run prompt.
- The overview renderer builds one Word-style table of contents and injects an
  `深入学习` action beside every H3 key concept or branch; tests assert the
  selected title is passed into the editable system-learning form.

## Residual risks

- Map-origin projects use the existing sidebar folder move interaction; no
  separate relationship or migration is required.
- Encyclopedia content quality still depends on the configured model, so future
  prompt changes should retain representative cross-discipline sampling.

## Documentation-system repair

The planning task exposed a forward-governance failure: the iteration contract
changed the long-lived project model, but the original process did not force a
landing-table audit. The repair is recorded in
`INCIDENT_REPORT_2026-07-14_LONG_TERM_DOC_SYNC.md` and the updated documentation
governance standard.

## Numbering migration

To reserve iteration 07 for this product contract without losing prior work:

- the delivered learning-rhythm and personalization slice moved from 07 to 08;
- the discipline-protocols and local-learning-lab discovery backlog moved from 08 to 09;
- their implementation and discovery status were otherwise preserved.
