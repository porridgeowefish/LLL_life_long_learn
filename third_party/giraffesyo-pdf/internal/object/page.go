package object

// Page returns the i-th page (1-based) by descending the page tree,
// guided by each /Pages node's /Count so subtrees with no room for page i
// are skipped. It returns null if the page cannot be located.
func (r *Reader) Page(i int) Value {
	if i < 1 {
		return Value{}
	}
	root := r.Trailer().Key("Root").Key("Pages")
	steps := 0
	return r.descend(root, i, &steps)
}

// descend walks a page-tree node looking for the want-th leaf beneath it.
func (r *Reader) descend(node Value, want int, steps *int) Value {
	if *steps++; *steps > maxPageWalk {
		return Value{}
	}
	switch node.Key("Type").Name() {
	case "Page":
		if want == 1 {
			return node
		}
		return Value{}
	case "Pages":
		kids := node.Key("Kids")
		for i := range kids.Len() {
			kid := kids.Index(i)
			n := kidLeafCount(kid)
			if want <= n {
				return r.descend(kid, want, steps)
			}
			want -= n
		}
	}
	return Value{}
}

// kidLeafCount returns a node's leaf-page count: /Count for a /Pages node,
// 1 for a /Page (or an untyped leaf).
func kidLeafCount(node Value) int {
	if node.Key("Type").Name() == "Pages" {
		if n, ok := node.Key("Count").Int64(); ok && n >= 0 {
			return int(n)
		}
		return 0
	}
	return 1
}

// Inherited resolves an attribute that a page may inherit from an
// ancestor /Pages node (Resources, MediaBox, …), walking /Parent with a
// bounded number of hops.
func Inherited(page Value, key string) Value {
	node := page
	for hops := 0; hops < maxPageWalk && !node.IsNull(); hops++ {
		if v := node.Key(key); !v.IsNull() {
			return v
		}
		node = node.Key("Parent")
	}
	return Value{}
}
