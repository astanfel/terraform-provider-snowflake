package sdk

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

type modifierType string

const (
	quoteModifierType   modifierType = "quotes"
	parenModifierType   modifierType = "paren"
	reverseModifierType modifierType = "reverse"
	equalsModifierType  modifierType = "equals"
)

type modifier interface {
	Modify(v any) string
}

type quoteModifier string

const (
	NoQuotes     quoteModifier = "no_quotes"
	DoubleQuotes quoteModifier = "double_quotes"
	SingleQuotes quoteModifier = "single_quotes"
)

func (qm quoteModifier) Modify(v any) string {
	s := fmt.Sprintf("%v", v)
	switch qm {
	case NoQuotes:
		return s
	case DoubleQuotes:
		escapedString := strings.ReplaceAll(s, qm.String(), qm.String()+qm.String())
		return fmt.Sprintf(`%v%v%v`, qm.String(), escapedString, qm.String())
	case SingleQuotes:
		// https://docs.snowflake.com/en/sql-reference/data-types-text#single-quoted-string-constants
		// replace all single quotes with \'
		escapedString := strings.ReplaceAll(s, qm.String(), `\'`)
		return fmt.Sprintf(`%v%v%v`, qm.String(), escapedString, qm.String())
	default:
		return s
	}
}

func (qm quoteModifier) String() string {
	switch qm {
	case NoQuotes:
		return ""
	case DoubleQuotes:
		return `"`
	case SingleQuotes:
		return `'`
	default:
		return ""
	}
}

type parenModifier string

const (
	NoParentheses parenModifier = "no_parentheses"
	Parentheses   parenModifier = "parentheses"
)

func (pm parenModifier) Modify(v any) string {
	s := fmt.Sprintf("%v", v)
	switch pm {
	case NoParentheses:
		return s
	case Parentheses:
		return fmt.Sprintf(`(%v)`, s)
	default:
		return s
	}
}

type reverseModifier string

const (
	NoReverse reverseModifier = "no_reverse"
	Reverse   reverseModifier = "reverse"
)

func (rm reverseModifier) Modify(v any) string {
	// v is []string{} type. result will be a joined string
	v = v.([]string)
	switch rm {
	case NoReverse:
		return strings.Join(v.([]string), " ")
	case Reverse:
		// reverse the order of the slice
		for i := len(v.([]string))/2 - 1; i >= 0; i-- {
			opp := len(v.([]string)) - 1 - i
			v.([]string)[i], v.([]string)[opp] = v.([]string)[opp], v.([]string)[i]
		}
		return strings.Join(v.([]string), " ")
	default:
		return strings.Join(v.([]string), " ")
	}
}

type equalsModifier string

const (
	Equals   equalsModifier = "equals"
	NoEquals equalsModifier = "no_equals"
)

func (em equalsModifier) Modify(v any) string {
	if em == Equals {
		return strings.TrimLeft(fmt.Sprintf(`%v = `, v), " ")
	}
	return strings.TrimLeft(fmt.Sprintf("%v ", v), " ")
}

func (b *sqlBuilder) getModifier(tag reflect.StructTag, tagName string, modType modifierType, defaultMod modifier) modifier {
	tagValue := strings.ToLower(tag.Get(tagName))
	if tagValue == "" {
		return defaultMod
	}
	parts := strings.Split(tagValue, ",")
	for _, part := range parts {
		if strings.Contains(part, string(modType)) {
			trimmedS := strings.TrimSpace(part)
			switch modType {
			case quoteModifierType:
				return quoteModifier(trimmedS)
			case parenModifierType:
				return parenModifier(trimmedS)
			case equalsModifierType:
				return equalsModifier(trimmedS)
			case reverseModifierType:
				return reverseModifier(trimmedS)
			}
		}
	}
	return defaultMod
}

func structToSQL(v interface{}) (string, error) {
	clauses, err := builder.parseStruct(v)
	if err != nil {
		return "", err
	}
	return builder.sql(clauses...), nil
}

const (
	builder sqlBuilder = "builder"
)

type sqlBuilder string

func (b sqlBuilder) renderStaticClause(clauses ...sqlClause) sqlClause {
	return sqlStaticClause(b.sql(clauses...))
}

// sql builds a SQL statement from sqlClauses.
func (b sqlBuilder) sql(clauses ...sqlClause) string {
	// remove nil and empty strings
	sList := make([]string, 0)
	for _, c := range clauses {
		if c != nil && c.String() != "" {
			sList = append(sList, c.String())
		}
	}

	return strings.Trim(strings.Join(sList, " "), " ")
}

// parseStruct parses a struct and returns a slice of sqlClauses.
func (b sqlBuilder) parseStruct(s interface{}) ([]sqlClause, error) {
	clauses := make([]sqlClause, 0)
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", v.Kind())
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Derefence pointers as long as they are not nil
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}

		switch value.Kind() {
		case reflect.Slice:
			sliceClause, err := b.parseFieldSlice(field, value)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, sliceClause)
			continue

		case reflect.Struct:
			fieldStructClause, err := b.parseFieldStruct(field, value)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, fieldStructClause)
			continue
		default:
			fieldClause, err := b.parseField(field, value)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, fieldClause)
		}
	}
	// prune all nil and empty string clauses
	prunedClauses := make([]sqlClause, 0)
	for _, c := range clauses {
		if c != nil && c.String() != "" {
			prunedClauses = append(prunedClauses, c)
		}
	}
	return prunedClauses, nil
}

func (b sqlBuilder) parseFieldStruct(field reflect.StructField, value reflect.Value) (sqlClause, error) {
	clauses := make([]sqlClause, 0)
	// all this does is check if the field has a keyword or is an identifier type before digging into struct
	ddlTag := field.Tag.Get("ddl")
	reflectedValue := b.getInterface(value)
	if ddlTag != "" {
		ddlTagParts := strings.Split(ddlTag, ",")
		ddlType := ddlTagParts[0]
		dbTag := field.Tag.Get("db")
		switch ddlType {
		case "keyword":
			clauses = append(clauses, sqlKeywordClause{
				key: dbTag,
				qm:  b.getModifier(field.Tag, "ddl", quoteModifierType, NoQuotes).(quoteModifier),
			})
		case "identifier":
			// identifiers are struct types but we don't want to dig into them
			if _, ok := reflectedValue.(Identifier); ok {
				if reflectedValue.(Identifier).Name() == "" {
					return nil, nil
				}
				return sqlIdentifierClause{
					key:   dbTag,
					value: reflectedValue.(Identifier),
					em:    b.getModifier(field.Tag, "ddl", equalsModifierType, NoEquals).(equalsModifier),
				}, nil
			}
		case "list":
			if dbTag != "" {
				clauses = append(clauses, sqlStaticClause(dbTag))
			}
			fieldStructClauses, err := b.parseStruct(reflectedValue)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, sqlListClause{
				clauses: fieldStructClauses,
				sep:     ",",
				pm:      b.getModifier(field.Tag, "ddl", parenModifierType, NoParentheses).(parenModifier),
			})
			return b.renderStaticClause(clauses...), nil
		}
	}
	fieldStructClauses, err := b.parseStruct(reflectedValue)
	if err != nil {
		return nil, err
	}
	clauses = append(clauses, fieldStructClauses...)
	return b.renderStaticClause(clauses...), nil
}

func (b sqlBuilder) parseFieldSlice(field reflect.StructField, value reflect.Value) (sqlClause, error) {
	// dereference any pointers
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	clauses := make([]sqlClause, 0)
	listClauses := make([]sqlClause, 0)
	// loop through the slice call parseStruct on each element (since the elements could be structs)
	for i := 0; i < value.Len(); i++ {
		reflectedValue := b.getInterface(value.Index(i))
		// test if reflectedValue is an Identifier. If it is it needs to be cast as an identifier
		identifier, ok := reflectedValue.(Identifier)
		if ok {
			listClauses = append(listClauses, sqlIdentifierClause{
				value: identifier,
				em:    b.getModifier(field.Tag, "ddl", equalsModifierType, NoEquals).(equalsModifier),
			})
			continue
		}
		// if it is a struct call parseStruct on it (recusive)
		if value.Index(i).Kind() == reflect.Struct || value.Index(i).Kind() == reflect.Ptr {
			structClauses, err := b.parseStruct(reflectedValue)
			if err != nil {
				return nil, err
			}
			// each element of the slice needs to be pre-rendered before the commas are added.
			sClause := b.renderStaticClause(structClauses...)
			listClauses = append(listClauses, sClause)
		} else {
			// if it is not a struct, then it is a primitive type and can be added directly.
			listClauses = append(listClauses, sqlStaticClause(fmt.Sprintf("%v", reflectedValue)))
		}
	}
	if len(listClauses) < 1 {
		return nil, nil
	}
	clauses = append(clauses, sqlListClause{
		clauses: listClauses,
		sep:     ",",
		pm:      b.getModifier(field.Tag, "ddl", parenModifierType, NoParentheses).(parenModifier),
	})
	sClause := b.renderStaticClause(clauses...)
	ddlTag := strings.Split(field.Tag.Get("ddl"), ",")[0]
	dbTag := field.Tag.Get("db")
	// depending on the ddl tag we may want to add a parameter clause or a keyword clause before rendered list clause
	switch ddlTag {
	case "parameter":
		return sqlParameterClause{
			key:   dbTag,
			value: sClause,
			qm:    b.getModifier(field.Tag, "ddl", quoteModifierType, NoQuotes).(quoteModifier),
			em:    b.getModifier(field.Tag, "ddl", equalsModifierType, Equals).(equalsModifier),
			rm:    b.getModifier(field.Tag, "ddl", reverseModifierType, NoReverse).(reverseModifier),
		}, nil
	case "keyword":
		return b.renderStaticClause(sqlKeywordClause{
			key: dbTag,
			qm:  b.getModifier(field.Tag, "ddl", quoteModifierType, NoQuotes).(quoteModifier),
		}, sClause), nil
	}
	return sClause, nil
}

// parseField parses an exported struct field and returns all nested sqlClauses.
func (b sqlBuilder) parseField(field reflect.StructField, value reflect.Value) (sqlClause, error) {
	// all fields needs a ddl tag otherwise we don't know what to do with them
	if field.Tag.Get("ddl") == "" {
		return nil, nil
	}

	clauses := make([]sqlClause, 0)
	var clause sqlClause

	// dereference any pointers
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	ddlTag := strings.Split(field.Tag.Get("ddl"), ",")[0]
	dbTag := field.Tag.Get("db")

	// static must be applied no matter what
	if ddlTag == "static" {
		return sqlStaticClause(dbTag), nil
	}

	if value.Kind() == reflect.Invalid {
		return nil, nil
	}
	reflectedValue := b.getInterface(value)

	// recurse into structs
	if field.Type.Kind() == reflect.Struct {
		structClauses, err := b.parseStruct(reflectedValue)
		if err != nil {
			return nil, err
		}
		return b.renderStaticClause(structClauses...), nil
	}

	if value.Kind() == reflect.Invalid {
		return nil, nil
	}

	switch ddlTag {
	case "keyword":
		if value.Kind() == reflect.Bool {
			useKeyword := reflectedValue.(bool)
			if useKeyword {
				clause = sqlKeywordClause{
					key: dbTag,
					qm:  b.getModifier(field.Tag, "ddl", quoteModifierType, NoQuotes).(quoteModifier),
				}
			} else {
				return nil, nil
			}
		} else {
			clause = sqlKeywordClause{
				key: reflectedValue,
				qm:  b.getModifier(field.Tag, "ddl", quoteModifierType, NoQuotes).(quoteModifier),
			}
		}
	case "identifier":
		clause = sqlIdentifierClause{
			key:   dbTag,
			value: reflectedValue.(Identifier),
			em:    b.getModifier(field.Tag, "ddl", equalsModifierType, NoEquals).(equalsModifier),
		}
	case "parameter":
		clause = sqlParameterClause{
			key:   dbTag,
			value: reflectedValue,
			em:    b.getModifier(field.Tag, "ddl", equalsModifierType, Equals).(equalsModifier),
			qm:    b.getModifier(field.Tag, "ddl", quoteModifierType, NoQuotes).(quoteModifier),
			rm:    b.getModifier(field.Tag, "ddl", reverseModifierType, NoReverse).(reverseModifier),
		}
	default:
		return nil, nil
	}
	return b.renderStaticClause(append(clauses, clause)...), nil
}

func (b sqlBuilder) getInterface(field reflect.Value) interface{} {
	// if the field is exported, then do this safely
	if field.CanInterface() {
		return field.Interface()
	}
	// otherwise yolo
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

type sqlListClause struct {
	clauses []sqlClause
	sep     string
	pm      parenModifier
}

func (v sqlListClause) String() string {
	if len(v.clauses) == 0 {
		return ""
	}
	clauseStrings := make([]string, len(v.clauses))
	for i, clause := range v.clauses {
		clauseStrings[i] = clause.String()
	}
	s := strings.Join(clauseStrings, v.sep)
	s = v.pm.Modify(s)
	return s
}

type sqlClause interface {
	String() string
}

type sqlStaticClause string

func (v sqlStaticClause) String() string {
	return string(v)
}

type sqlKeywordClause struct {
	key interface{}
	qm  quoteModifier
}

func (v sqlKeywordClause) String() string {
	return v.qm.Modify(v.key)
}

type sqlIdentifierClause struct {
	key   string
	value Identifier
	em    equalsModifier
}

func (v sqlIdentifierClause) String() string {
	var name string
	// object identifiers need to be fully qualified
	if _, ok := v.value.(ObjectIdentifier); ok {
		name = v.value.(ObjectIdentifier).FullyQualifiedName()
	} else {
		name = DoubleQuotes.Modify(v.value.Name())
	}
	// else try to get the string value
	if v.key != "" {
		return v.em.Modify(v.key) + name
	}
	return name
}

type sqlParameterClause struct {
	key   string
	value interface{}

	// modifiers
	qm quoteModifier
	em equalsModifier
	rm reverseModifier
}

func (v sqlParameterClause) String() string {
	// the reverse modifier is never used with equals modifier, so we just ignore it
	if v.rm == Reverse {
		// "value" key
		return v.rm.Modify([]string{v.key, v.qm.Modify(v.value)})
	}
	// key =
	s := v.em.Modify(v.key)
	if v.value == nil {
		return s
	}
	// key = "value"
	s += v.qm.Modify(v.value)
	return s
}
