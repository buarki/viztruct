// Package rewriter reorders fields of a named struct in-place, using the
// optimized ordering computed by the structi analyzer. It preserves comments
// and struct tags via github.com/dave/dst.
//
// The rewriter deliberately refuses to touch structs with shapes that could
// change semantics or confuse the user. See UnsupportedError for the list.
package rewriter

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"

	"github.com/buarki/viztruct/structi"
)

// UnsupportedError is returned when the target struct has a shape the rewriter
// refuses to handle (grouped decls, generics, embedded pointer, etc.). The
// caller should surface Reason to the user verbatim.
type UnsupportedError struct {
	Struct string
	Reason string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("cannot rewrite %s: %s", e.Struct, e.Reason)
}

// RewriteStruct returns the new contents of filePath with the named struct's
// fields reordered to the optimal layout. The original file on disk is not
// modified; callers apply the returned bytes however they like (write to disk,
// pipe through a VSCode WorkspaceEdit, etc.).
func RewriteStruct(filePath, structName string) ([]byte, error) {
	pkgDir := filepath.Dir(filePath)

	groups, err := structi.AnalyseStructsAtDirectoryPath(pkgDir, nil)
	if err != nil {
		return nil, fmt.Errorf("analyze package: %w", err)
	}

	var info *structi.Info
	for _, g := range groups {
		for i := range g {
			if g[i].Name == structName {
				info = &g[i]
				break
			}
		}
		if info != nil {
			break
		}
	}
	if info == nil {
		return nil, fmt.Errorf("struct %s not found in %s", structName, pkgDir)
	}
	if info.OriginalSize == info.OptimizedSize {
		return nil, &UnsupportedError{
			Struct: structName,
			Reason: "struct is already optimally laid out; no rewrite needed",
		}
	}

	optimizedOrder := extractFieldOrder(info.OptimizedFields)
	if len(optimizedOrder) == 0 {
		return nil, &UnsupportedError{
			Struct: structName,
			Reason: "optimizer did not return a usable field ordering",
		}
	}

	file, err := parseDSTFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	target, err := findStructType(file, structName)
	if err != nil {
		return nil, err
	}

	if err := validateStructShape(structName, target); err != nil {
		return nil, err
	}

	if err := reorderFields(target, optimizedOrder); err != nil {
		return nil, &UnsupportedError{Struct: structName, Reason: err.Error()}
	}

	var out bytes.Buffer
	if err := decorator.Fprint(&out, file); err != nil {
		return nil, fmt.Errorf("reprint: %w", err)
	}
	return out.Bytes(), nil
}

func extractFieldOrder(fields []structi.Field) []string {
	order := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.IsPadding {
			continue
		}
		order = append(order, f.Name)
	}
	return order
}

// parseDSTFile parses filePath into a dst.File with comments preserved.
func parseDSTFile(filePath string) (*dst.File, error) {
	d := decorator.NewDecorator(nil)
	return d.ParseFile(filePath, nil, 0)
}

// findStructType locates the TopLevel TypeSpec for the named struct.
// Only top-level `type X struct { ... }` declarations are supported in MVP.
func findStructType(file *dst.File, structName string) (*dst.StructType, error) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*dst.GenDecl)
		if !ok || gen.Tok.String() != "type" {
			continue
		}
		// Reject `type ( ... )` blocks to keep MVP scope tight.
		if gen.Lparen && len(gen.Specs) > 1 {
			for _, spec := range gen.Specs {
				ts, ok := spec.(*dst.TypeSpec)
				if !ok || ts.Name == nil || ts.Name.Name != structName {
					continue
				}
				return nil, &UnsupportedError{
					Struct: structName,
					Reason: "struct is declared inside a `type (...)` block; move it to its own `type` declaration to enable rewrite",
				}
			}
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*dst.TypeSpec)
			if !ok || ts.Name == nil || ts.Name.Name != structName {
				continue
			}
			if ts.TypeParams != nil && len(ts.TypeParams.List) > 0 {
				return nil, &UnsupportedError{
					Struct: structName,
					Reason: "generic structs are not supported by the rewriter yet",
				}
			}
			st, ok := ts.Type.(*dst.StructType)
			if !ok {
				return nil, &UnsupportedError{
					Struct: structName,
					Reason: "declaration exists but is not a struct type",
				}
			}
			return st, nil
		}
	}
	return nil, fmt.Errorf("top-level `type %s struct` declaration not found", structName)
}

// validateStructShape rejects structs containing features the MVP rewriter
// cannot safely handle. Each rejection produces an UnsupportedError with a
// human-readable reason.
func validateStructShape(structName string, st *dst.StructType) error {
	if st.Fields == nil {
		return nil
	}
	for _, f := range st.Fields.List {
		if len(f.Names) > 1 {
			return &UnsupportedError{
				Struct: structName,
				Reason: fmt.Sprintf("field %q uses a grouped declaration (`a, b %s`); split it into separate lines to enable rewrite",
					f.Names[0].Name, exprText(f.Type)),
			}
		}
		// Reject pointer-embedded fields: `*Foo` without an explicit name. Their
		// "name" in Go is the referenced type, which is easy to match, but keeping
		// scope tight avoids edge cases around nil-vs-zero semantics.
		if len(f.Names) == 0 {
			if _, ok := f.Type.(*dst.StarExpr); ok {
				return &UnsupportedError{
					Struct: structName,
					Reason: "pointer-embedded fields are not supported by the rewriter yet",
				}
			}
		}
		// //go:... directives on individual fields carry semantic meaning.
		for _, c := range f.Decorations().Start.All() {
			if strings.HasPrefix(c, "//go:") {
				return &UnsupportedError{
					Struct: structName,
					Reason: fmt.Sprintf("field has a `//go:` directive (%q); rewriter refuses to move it", c),
				}
			}
		}
	}
	return nil
}

// reorderFields rearranges st.Fields.List to match the order of names in
// `order`. Any field present in the struct but absent from `order` is an
// error (it means the analyzer and the AST disagree about what's there).
func reorderFields(st *dst.StructType, order []string) error {
	if st.Fields == nil || len(st.Fields.List) == 0 {
		return nil
	}

	byName := make(map[string]*dst.Field, len(st.Fields.List))
	for _, f := range st.Fields.List {
		name, err := fieldName(f)
		if err != nil {
			return err
		}
		if _, dup := byName[name]; dup {
			return fmt.Errorf("duplicate field name %q in struct", name)
		}
		byName[name] = f
	}

	reordered := make([]*dst.Field, 0, len(st.Fields.List))
	used := make(map[string]bool, len(order))
	for _, name := range order {
		f, ok := byName[name]
		if !ok {
			return fmt.Errorf("analyzer reported field %q that is not present in the source", name)
		}
		if used[name] {
			return fmt.Errorf("analyzer listed field %q more than once", name)
		}
		used[name] = true
		reordered = append(reordered, f)
	}
	if len(reordered) != len(st.Fields.List) {
		var missing []string
		for name := range byName {
			if !used[name] {
				missing = append(missing, name)
			}
		}
		return fmt.Errorf("analyzer omitted fields from optimized order: %v", missing)
	}

	st.Fields.List = reordered
	return nil
}

// fieldName returns the single-identifier name used to match a dst.Field
// against the analyzer's view. For embedded fields it unwraps *dst.Ident and
// *dst.SelectorExpr; unsupported shapes return an error.
func fieldName(f *dst.Field) (string, error) {
	if len(f.Names) == 1 {
		return f.Names[0].Name, nil
	}
	if len(f.Names) > 1 {
		return "", fmt.Errorf("grouped field declaration not supported")
	}
	switch t := f.Type.(type) {
	case *dst.Ident:
		return t.Name, nil
	case *dst.SelectorExpr:
		if t.Sel != nil {
			return t.Sel.Name, nil
		}
	case *dst.StarExpr:
		return "", fmt.Errorf("pointer-embedded field not supported")
	case *dst.IndexExpr, *dst.IndexListExpr:
		return "", fmt.Errorf("generic-embedded field not supported")
	}
	return "", fmt.Errorf("unrecognized embedded field shape")
}

// exprText returns a short, readable representation of a type expression for
// use in error messages. Avoids pulling in decorator.Fprint for each error.
func exprText(e dst.Expr) string {
	switch v := e.(type) {
	case *dst.Ident:
		return v.Name
	case *dst.SelectorExpr:
		if id, ok := v.X.(*dst.Ident); ok && v.Sel != nil {
			return id.Name + "." + v.Sel.Name
		}
	case *dst.StarExpr:
		return "*" + exprText(v.X)
	}
	return "<expr>"
}
