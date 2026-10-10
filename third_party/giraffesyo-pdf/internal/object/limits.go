package object

// Resource limits guarding against hostile files. They are vars, not
// consts, so tests can lower them.
var (
	maxObjects     = 1 << 22 // xref entries / highest object number
	maxPrevChain   = 64      // xref /Prev + /XRefStm hops
	maxObjStmObjs  = 100_000 // objects packed in one object stream
	maxObjStmCache = 4       // decoded object streams held at once
	maxParseDepth  = 64      // nested dict/array depth in the parser
	maxRefHops     = 32      // indirect-reference chain length
	maxFilterChain = 8       // filters in one /Filter array
	maxPageWalk    = 100_000 // page-tree descent / inheritance steps
)
