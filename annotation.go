package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dave/dst"
	log "github.com/sirupsen/logrus"
)

const annotationPrefix = "// __golines:shorten:"
const combineAnnotationSuffix = "// __golines:combine"

// CreateCombineAnnotation generates the text of a comment that will annotate lines to be combined
func CreateCombineAnnotation() string {
	return fmt.Sprintf("%s", combineAnnotationSuffix)
}

// CreateAnnotation generates the text of a comment that will annotate long lines.
func CreateAnnotation(length int) string {
	return fmt.Sprintf(
		"%s%d",
		annotationPrefix,
		length,
	)
}

func IsCombineAnnotation(line string) bool {
	return strings.HasSuffix(line, combineAnnotationSuffix)
}

// IsAnnotation determines whether the given line is an annotation created with CreateAnnotation.
func IsAnnotation(line string) bool {
	return strings.HasPrefix(
		strings.Trim(line, " \t"),
		annotationPrefix,
	)
}

func HasCombineAnnotation(node dst.Node) bool {
	startDecorations := node.Decorations().Start.All()
	return len(startDecorations) > 0 && IsCombineAnnotation(startDecorations[len(startDecorations)-1])
}

func RemoveCombineSuffix(line string) string {
	trimmed := strings.TrimSuffix(line, combineAnnotationSuffix)
	return trimmed
}

func RemoveCombineAnnotations(node dst.Node) {
	for i, dec := range node.Decorations().Start.All() {
		if IsCombineAnnotation(dec) {
			node.Decorations().Start = append(node.Decorations().Start[:i], node.Decorations().Start[i+1:]...)
			break
		}
	}
	for i, dec := range node.Decorations().End.All() {
		if IsCombineAnnotation(dec) {
			node.Decorations().End = append(node.Decorations().End[:i], node.Decorations().End[i+1:]...)
			break
		}
	}
	switch n := node.(type) {
	case *dst.BinaryExpr:
		RemoveCombineAnnotations(n.X)
		RemoveCombineAnnotations(n.Y)
	}
}

// HasAnnotation determines whether the given AST node has a line length annotation on it.
func HasAnnotation(node dst.Node) bool {
	startDecorations := node.Decorations().Start.All()
	return len(startDecorations) > 0 &&
		IsAnnotation(startDecorations[len(startDecorations)-1])
}

// HasTailAnnotation determines whether the given AST node has a line length annotation at its
// end. This is needed to catch long function declarations with inline interface definitions.
func HasTailAnnotation(node dst.Node) bool {
	endDecorations := node.Decorations().End.All()
	return len(endDecorations) > 0 &&
		IsAnnotation(endDecorations[len(endDecorations)-1])
}

// HasAnnotationRecursive determines whether the given node or one of its children has a
// golines annotation on it. It's currently implemented for function declarations, fields,
// call expressions, and selector expressions only.
func HasAnnotationRecursive(node dst.Node) bool {
	if HasAnnotation(node) {
		return true
	}

	switch n := node.(type) {
	case *dst.GenDecl:
		for _, spec := range n.Specs {
			if HasAnnotationRecursive(spec) {
				return true
			}
		}
	case *dst.FuncDecl:
		if n.Type != nil && n.Type.Params != nil {
			for _, item := range n.Type.Params.List {
				if HasAnnotationRecursive(item) {
					return true
				}
			}
		}
	case *dst.StructType:
		return HasAnnotationRecursive(n.Fields)
	case *dst.FuncType:
		hasAny := n.Params != nil && HasAnnotationRecursive(n.Params)
		hasAny = hasAny || (n.TypeParams != nil && HasAnnotationRecursive(n.TypeParams))
		hasAny = hasAny || (n.Results != nil && HasAnnotationRecursive(n.Results))
		return hasAny
	case *dst.TypeSpec:
		return HasAnnotationRecursive(n.Type)
	case *dst.Field:
		return HasTailAnnotation(n) || HasAnnotationRecursive(n.Type)
	case *dst.FieldList:
		for _, field := range n.List {
			if HasAnnotationRecursive(field) {
				return true
			}
		}
	case *dst.SelectorExpr:
		return HasAnnotation(n.Sel) || HasAnnotation(n.X)
	case *dst.CallExpr:
		if HasAnnotationRecursive(n.Fun) {
			return true
		}

		for _, arg := range n.Args {
			if HasAnnotation(arg) {
				return true
			}
		}
	case *dst.InterfaceType:
		for _, field := range n.Methods.List {
			if HasAnnotationRecursive(field) {
				return true
			}
		}
	default:
		log.Debugf("Couldn't analyze type for annotations: %+v", n)
	}

	return false
}

// ParseAnnotation returns the line length encoded in a golines annotation. If none is found,
// it returns -1.
func ParseAnnotation(line string) int {
	if IsAnnotation(line) {
		components := strings.SplitN(line, ":", 3)
		val, err := strconv.Atoi(components[2])
		if err != nil {
			return -1
		}
		return val
	}
	return -1
}
