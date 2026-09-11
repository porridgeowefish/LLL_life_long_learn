# Iteration 16 Acceptance Criteria

1. A supported source parses asynchronously to exactly one `content.md`; Sources
   shows original filename/type/status only and no derived-content preview.
2. Composer citation selects only ready parsed sources, previews content in its
   own compact panel, and sends selected references with the next teacher turn.
3. A map-origin `teachingOutline` is persisted and appears in the teacher system
   context; it names the teaching boundary without overriding learner intent.
4. Confirmed consolidation updates Intro and Body from sealed conversation
   evidence. Practice remains unchanged unless the task objective explicitly
   marks `practiceRequested` in the approved task.
5. The generated Body is a standalone teaching manuscript with a critical
   thinking conclusion, not a transcript or special Summary artifact.
6. Teacher usage records persist input/output tokens by response and a paginated
   token page aggregates them by learning-unit conversation. No assistant usage
   is exposed.
7. No standalone Agent-management navigation or frontend route remains.
8. Unit, integration, frontend, production-build, and repository quality gates
   pass; browser E2E remains learner-run.
