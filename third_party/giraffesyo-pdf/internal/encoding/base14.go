package encoding

// symbolEncodingTable is the built-in encoding of the PDF base-14 Symbol
// font. Glyph names come from Adobe's Symbol encoding and are resolved through
// the package's embedded Adobe Glyph List.
func symbolEncodingTable() [256]rune {
	var table [256]rune
	setNames(&table, 32, []string{
		"space", "exclam", "universal", "numbersign", "existential", "percent", "ampersand", "suchthat",
		"parenleft", "parenright", "asteriskmath", "plus", "comma", "minus", "period", "slash",
		"zero", "one", "two", "three", "four", "five", "six", "seven",
		"eight", "nine", "colon", "semicolon", "less", "equal", "greater", "question",
		"congruent", "Alpha", "Beta", "Chi", "Delta", "Epsilon", "Phi", "Gamma",
		"Eta", "Iota", "theta1", "Kappa", "Lambda", "Mu", "Nu", "Omicron",
		"Pi", "Theta", "Rho", "Sigma", "Tau", "Upsilon", "sigma1", "Omega",
		"Xi", "Psi", "Zeta", "bracketleft", "therefore", "bracketright", "perpendicular", "underscore",
		"radicalex", "alpha", "beta", "chi", "delta", "epsilon", "phi", "gamma",
		"eta", "iota", "phi1", "kappa", "lambda", "mu", "nu", "omicron",
		"pi", "theta", "rho", "sigma", "tau", "upsilon", "omega1", "omega",
		"xi", "psi", "zeta", "braceleft", "bar", "braceright", "similar",
	})
	setNames(&table, 161, []string{
		"Upsilon1", "minute", "lessequal", "fraction", "infinity", "florin", "club", "diamond",
		"heart", "spade", "arrowboth", "arrowleft", "arrowup", "arrowright", "arrowdown", "degree",
		"plusminus", "second", "greaterequal", "multiply", "proportional", "partialdiff", "bullet", "divide",
		"notequal", "equivalence", "approxequal", "ellipsis", "arrowvertex", "arrowhorizex", "carriagereturn", "aleph",
		"Ifraktur", "Rfraktur", "weierstrass", "circlemultiply", "circleplus", "emptyset", "intersection", "union",
		"propersuperset", "reflexsuperset", "notsubset", "propersubset", "reflexsubset", "element", "notelement", "angle",
		"gradient", "registerserif", "copyrightserif", "trademarkserif", "product", "radical", "dotmath", "logicalnot",
		"logicaland", "logicalor", "arrowdblboth", "arrowdblleft", "arrowdblup", "arrowdblright", "arrowdbldown", "lozenge",
		"angleleft", "registersans", "copyrightsans", "trademarksans", "summation", "parenlefttp", "parenleftex", "parenleftbt",
		"bracketlefttp", "bracketleftex", "bracketleftbt", "bracelefttp", "braceleftmid", "braceleftbt", "braceex",
	})
	setNames(&table, 241, []string{
		"angleright", "integral", "integraltp", "integralex", "integralbt", "parenrighttp", "parenrightex",
		"parenrightbt", "bracketrighttp", "bracketrightex", "bracketrightbt", "bracerighttp",
		"bracerightmid", "bracerightbt",
	})
	return table
}

func setNames(table *[256]rune, start int, names []string) {
	for i, name := range names {
		runes := []rune(GlyphToText(name))
		if len(runes) == 1 {
			table[start+i] = runes[0]
		}
	}
}

// zapfDingbatsTable maps the base-14 ZapfDingbats encoding to modern Unicode.
func zapfDingbatsTable() [256]rune {
	var table [256]rune
	table[0x20] = ' '
	setRuneRange(&table, 0x21, 0x24, 0x2701)
	table[0x25] = 0x260e
	setRuneRange(&table, 0x26, 0x29, 0x2706)
	table[0x2a] = 0x261b
	table[0x2b] = 0x261e
	setRuneRange(&table, 0x2c, 0x47, 0x270c)
	table[0x48] = 0x2605
	setRuneRange(&table, 0x49, 0x6b, 0x2729)
	table[0x6c] = 0x25cf
	table[0x6d] = 0x274d
	table[0x6e] = 0x25a0
	setRuneRange(&table, 0x6f, 0x72, 0x274f)
	table[0x73] = 0x25b2
	table[0x74] = 0x25bc
	table[0x75] = 0x25c6
	table[0x76] = 0x2756
	table[0x77] = 0x25d7
	setRuneRange(&table, 0x78, 0x7e, 0x2758)
	setRuneRange(&table, 0xa1, 0xa7, 0x2761)
	table[0xa8] = 0x2663
	table[0xa9] = 0x2666
	table[0xaa] = 0x2665
	table[0xab] = 0x2660
	setRuneRange(&table, 0xac, 0xb5, 0x2460)
	setRuneRange(&table, 0xb6, 0xd4, 0x2776)
	table[0xd5] = 0x2192
	table[0xd6] = 0x2194
	table[0xd7] = 0x2195
	setRuneRange(&table, 0xd8, 0xef, 0x2798)
	setRuneRange(&table, 0xf1, 0xfe, 0x27b1)
	return table
}

func setRuneRange(table *[256]rune, first, last byte, firstRune rune) {
	for code, r := first, firstRune; ; code, r = code+1, r+1 {
		table[code] = r
		if code == last {
			return
		}
	}
}
