package db

import (
	"strings"

	"sqlite-explorer/backend/apperrors"
)

// tokenKind classifies the lexical tokens the statement splitter and classifier care about.
type tokenKind int

const (
	tokWord   tokenKind = iota // bare identifier or keyword
	tokQuoted                  // "ident", `ident`, or [ident]
	tokString                  // 'text'
	tokSemi
	tokLParen
	tokRParen
	tokComma
	tokOther // operators, numbers, parameter markers
)

type token struct {
	kind  tokenKind
	text  string
	start int
	end   int
}

// keyword returns the ASCII-uppercased text of a bare word, or "" for any other token.
// SQLite keywords are ASCII-only, so Unicode case folding must not apply here.
func (t token) keyword() string {
	if t.kind != tokWord {
		return ""
	}
	return asciiUpper(t.text)
}

func (t token) is(kw string) bool {
	return t.kind == tokWord && asciiUpper(t.text) == kw
}

// statement is one SQL statement with its source text (without the terminating semicolon).
type statement struct {
	text   string
	tokens []token
}

// tokenize splits sql into tokens using SQLite's lexical rules for whitespace, comments,
// quoting, and identifiers. Whitespace and comments are dropped.
func tokenize(sql string) ([]token, error) {
	tokens, _, err := lex(sql)
	return tokens, err
}

// lex is tokenize that also reports whether the input ends inside a block comment.
func lex(sql string) (tokens []token, openComment bool, err error) {
	if strings.IndexByte(sql, 0) >= 0 {
		return nil, false, apperrors.New(apperrors.CodeMalformedSQL, "SQL must not contain NUL characters.", "")
	}
	i := 0
	for i < len(sql) {
		c := sql[i]
		start := i
		switch {
		case isSQLSpace(c):
			i++
			continue
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			continue
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			// An unterminated block comment runs to the end of input, as in SQLite.
			end := strings.Index(sql[i+2:], "*/")
			if end < 0 {
				i = len(sql)
				openComment = true
			} else {
				i += 2 + end + 2
			}
			continue
		case c == '\'' || c == '"' || c == '`':
			end, ok := scanQuoted(sql, i, c)
			if !ok {
				return nil, false, unterminated(sql[start:])
			}
			i = end
			kind := tokQuoted
			if c == '\'' {
				kind = tokString
			}
			tokens = append(tokens, token{kind: kind, text: sql[start:i], start: start, end: i})
			continue
		case c == '[':
			end := strings.IndexByte(sql[i+1:], ']')
			if end < 0 {
				return nil, false, unterminated(sql[start:])
			}
			i += 1 + end + 1
			tokens = append(tokens, token{kind: tokQuoted, text: sql[start:i], start: start, end: i})
			continue
		case c == ';':
			i++
			tokens = append(tokens, token{kind: tokSemi, text: ";", start: start, end: i})
			continue
		case c == '(':
			i++
			tokens = append(tokens, token{kind: tokLParen, text: "(", start: start, end: i})
			continue
		case c == ')':
			i++
			tokens = append(tokens, token{kind: tokRParen, text: ")", start: start, end: i})
			continue
		case c == ',':
			i++
			tokens = append(tokens, token{kind: tokComma, text: ",", start: start, end: i})
			continue
		case isIDChar(c):
			// Like sqlite3_complete(), treat any run of identifier characters as one token;
			// only runs that start with a letter can be keywords.
			for i < len(sql) && isIDChar(sql[i]) {
				i++
			}
			kind := tokWord
			if isDigit(c) || c == '$' {
				kind = tokOther
			}
			tokens = append(tokens, token{kind: kind, text: sql[start:i], start: start, end: i})
			continue
		default:
			i++
		}
		tokens = append(tokens, token{kind: tokOther, text: sql[start:i], start: start, end: i})
	}
	return tokens, openComment, nil
}

// scanQuoted returns the index just past a quoted token starting at sql[start] == quote.
// A doubled quote character is an escaped quote.
func scanQuoted(sql string, start int, quote byte) (int, bool) {
	i := start + 1
	for i < len(sql) {
		if sql[i] == quote {
			if i+1 < len(sql) && sql[i+1] == quote {
				i += 2
				continue
			}
			return i + 1, true
		}
		i++
	}
	return 0, false
}

func unterminated(rest string) error {
	if len(rest) > 40 {
		rest = rest[:40] + "..."
	}
	return apperrors.New(apperrors.CodeMalformedSQL, "The SQL has an unterminated quote or bracket.", rest)
}

// splitState is the state machine of SQLite's sqlite3_complete(). A semicolon ends a
// statement except inside a CREATE TRIGGER body, which ends only at "; END ;".
type splitState int

const (
	stInvalid splitState = iota // nothing seen yet
	stStart                     // at a statement boundary
	stNormal
	stExplain
	stCreate
	stTrigger
	stSemi
	stEnd
)

func (s splitState) next(tok token) splitState {
	if tok.kind == tokSemi {
		switch s {
		case stTrigger, stSemi:
			return stSemi
		default:
			return stStart
		}
	}
	switch s {
	case stInvalid, stStart:
		switch {
		case tok.is("EXPLAIN"):
			return stExplain
		case tok.is("CREATE"):
			return stCreate
		}
		return stNormal
	case stExplain:
		switch {
		case tok.is("CREATE"):
			return stCreate
		case tok.is("EXPLAIN"), tok.is("TEMP"), tok.is("TEMPORARY"), tok.is("TRIGGER"), tok.is("END"):
			return stNormal
		}
		return stExplain
	case stCreate:
		switch {
		case tok.is("TEMP"), tok.is("TEMPORARY"):
			return stCreate
		case tok.is("TRIGGER"):
			return stTrigger
		}
		return stNormal
	case stSemi:
		if tok.is("END") {
			return stEnd
		}
		return stTrigger
	case stEnd:
		return stTrigger
	}
	return s
}

// splitStatements splits sql into statements, dropping empty ones.
func splitStatements(sql string) ([]statement, error) {
	tokens, err := tokenize(sql)
	if err != nil {
		return nil, err
	}

	var stmts []statement
	var current []token
	state := stInvalid
	for _, tok := range tokens {
		state = state.next(tok)
		if tok.kind == tokSemi && state == stStart {
			if len(current) > 0 {
				text := sql[current[0].start:current[len(current)-1].end]
				stmts = append(stmts, statement{text: text, tokens: current})
			}
			current = nil
			continue
		}
		current = append(current, tok)
	}
	if len(current) > 0 {
		text := sql[current[0].start:current[len(current)-1].end]
		stmts = append(stmts, statement{text: text, tokens: current})
	}
	return stmts, nil
}

func isSQLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isIDChar follows SQLite: bytes >= 0x80 are identifier characters.
func isIDChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || isDigit(c) || c == '_' || c == '$' || c >= 0x80
}

func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}
