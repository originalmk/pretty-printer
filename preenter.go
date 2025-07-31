package preenter

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type PrettyPrinter struct {
	options     SprintOptions
	visitedPtrs map[any]bool
}

func DefaultPrettyPrinter() PrettyPrinter {
	return PrettyPrinter{
		options:     DefaultSprintOptions(),
		visitedPtrs: make(map[any]bool),
	}
}

type SprintOptions struct {
	skipHeader  bool
	forceIndent bool
}

func DefaultSprintOptions() SprintOptions {
	return SprintOptions{
		skipHeader: false,
	}
}

type AnnotatedStruct struct {
	HasTitleField   bool
	TitleField      AnnotatedField
	AnnotatedFields AnnotatedFields
}

type AnnotatedFields = map[string]AnnotatedField

type AnnotatedField struct {
	Name  string
	Value reflect.Value
	Tags  ParsedTags
}

type ParsedTags struct {
	isTitle    bool
	orderIndex uint
}

func annotateStruct(v any) (AnnotatedStruct, error) {
	annotatedFields, err := annotateFields(v)
	if err != nil {
		return AnnotatedStruct{}, nil
	}

	var hasTitleField bool
	var titleField AnnotatedField

	for _, field := range annotatedFields {
		if field.Tags.isTitle {
			hasTitleField = true
			titleField = field
			break
		}
	}

	return AnnotatedStruct{
		HasTitleField:   hasTitleField,
		TitleField:      titleField,
		AnnotatedFields: annotatedFields,
	}, nil
}

func annotateFields(v any) (AnnotatedFields, error) {
	result := make(AnnotatedFields)
	vType := reflect.TypeOf(v)
	vFields := reflect.VisibleFields(vType)
	vValue := reflect.ValueOf(v)

	for _, field := range vFields {
		parsedTags, err := fieldParsedTags(field)
		if err != nil {
			return nil, err
		}

		result[field.Name] = AnnotatedField{
			Name:  field.Name,
			Value: vValue.FieldByName(field.Name),
			Tags:  parsedTags,
		}
	}

	return result, nil
}

func fieldParsedTags(field reflect.StructField) (ParsedTags, error) {
	result := ParsedTags{}

	rawTags, err := fieldRawTags(field)
	if err != nil {
		return result, err
	}

	semTag, ok := rawTags["sem"]
	if ok {
		result.isTitle = (semTag == "title")
	}

	ordTag, ok := rawTags["ord"]
	if ok {
		orderIndex, _ := strconv.Atoi(ordTag)
		result.orderIndex = uint(orderIndex)
	}

	return result, nil
}

func fieldRawTags(field reflect.StructField) (map[string]string, error) {
	result := make(map[string]string)
	prettyTag := field.Tag.Get("pretty")

	if prettyTag != "" {
		for tagElement := range strings.SplitSeq(prettyTag, ",") {
			elementSplit := strings.Split(tagElement, "=")

			if len(elementSplit) != 2 {
				return result, errors.New("invalid tag")
			}

			elementKey := elementSplit[0]
			elementValue := elementSplit[1]

			result[elementKey] = elementValue
		}
	}

	return result, nil
}

func (PrettyPrinter) sprintPrimitive(v any) (string, error) {
	var supportedKinds = []reflect.Kind{
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.String,
	}

	vKind := reflect.TypeOf(v).Kind()
	supported := slices.Contains(supportedKinds, vKind)
	if !supported {
		return "", fmt.Errorf("unsupported kind: %s", vKind)
	}

	return fmt.Sprintf("%v", v), nil
}

func (pp PrettyPrinter) sprintSlice(v any) (string, error) {
	if reflect.TypeOf(v).Kind() != reflect.Slice {
		return "", errors.New("kind is not slice")
	}

	var resultBuilder strings.Builder

	if !pp.options.skipHeader {
		resultBuilder.WriteString("List")
		resultBuilder.WriteString("\n")
	}

	vValue := reflect.ValueOf(v)

	var listItemsBuilder strings.Builder

	for i := 0; i < vValue.Len(); i++ {
		prevForceIndent := pp.options.forceIndent
		pp.options.forceIndent = false
		s, err := pp.SprintPretty(vValue.Index(i).Interface())
		pp.options.forceIndent = prevForceIndent
		if err != nil {
			return "", err
		}

		sLines := strings.Split(s, "\n")
		sLinesCount := len(sLines)

		if sLinesCount == 0 {
			return "", errors.New("list element serialized to empty string")
		}

		// Number of lines must be greater than 0
		partialResult := fmt.Sprintf("* %s", sLines[0])
		partialResult = strings.Join(
			append([]string{partialResult}, sLines[1:]...), "\n  ")

		listItemsBuilder.WriteString(partialResult)
		listItemsBuilder.WriteString("\n")
	}

	if vValue.Len() == 0 {
		listItemsBuilder.WriteString("... empty list ...")
	}

	listItemsSprint := listItemsBuilder.String()
	listItemsIndented := sprintIndent(listItemsSprint, " ", 4)
	resultBuilder.WriteString(listItemsIndented)

	lastEOLTrimmed := strings.TrimRight(resultBuilder.String(), "\n")

	return lastEOLTrimmed, nil
}

func (pp PrettyPrinter) sprintStruct(v any) (string, error) {
	if reflect.TypeOf(v).Kind() != reflect.Struct {
		return "", errors.New("kind is not struct")
	}

	annotated, err := annotateStruct(v)
	if err != nil {
		return "", err
	}

	var resultBuilder strings.Builder

	if annotated.HasTitleField {
		prettyPrimitive, err := pp.sprintPrimitive(annotated.TitleField.Value.Interface())
		if err != nil {
			return "", err
		}

		resultBuilder.WriteString(prettyPrimitive)
		resultBuilder.WriteString("\n")
	} else if !pp.options.skipHeader {
		resultBuilder.WriteString(reflect.TypeOf(v).Name())
		resultBuilder.WriteString("\n")
	}

	fieldKeys := slices.Collect(maps.Keys(annotated.AnnotatedFields))
	slices.SortFunc(fieldKeys, func(a string, b string) int {
		aVal := annotated.AnnotatedFields[a].Tags.orderIndex
		bVal := annotated.AnnotatedFields[b].Tags.orderIndex

		if aVal == 0 && bVal != 0 {
			return 1
		}

		if bVal == 0 && aVal != 0 {
			return -1
		}

		return int(aVal - bVal)
	})

	for _, fieldKey := range fieldKeys {
		field := annotated.AnnotatedFields[fieldKey]

		if annotated.TitleField.Name == field.Name {
			continue
		}

		var fieldBuilder strings.Builder

		prevSkipHeader := pp.options.skipHeader
		pp.options.skipHeader = true
		prevForceIndent := pp.options.forceIndent
		pp.options.forceIndent = true
		innerSprint, err := pp.SprintPretty(field.Value.Interface())
		if err != nil {
			return "", err
		}
		pp.options.skipHeader = prevSkipHeader
		pp.options.forceIndent = prevForceIndent

		innerKind := field.Value.Kind()

		fieldBuilder.WriteString(field.Name)

		if len(strings.Split(innerSprint, "\n")) > 1 || innerKind == reflect.Struct || innerKind == reflect.Slice {
			fieldBuilder.WriteString(" >>>\n")
		} else {
			fieldBuilder.WriteString(" = ")
		}

		fieldBuilder.WriteString(innerSprint)

		fieldSprint := fieldBuilder.String()

		fmt.Println(pp.options.forceIndent)
		if !pp.options.skipHeader || pp.options.forceIndent {
			fieldSprint = sprintIndent(fieldSprint, " ", 4)
		}

		resultBuilder.WriteString(fieldSprint)
		resultBuilder.WriteString("\n")
	}

	lastEOLTrimmed := strings.TrimRight(resultBuilder.String(), "\n")

	return lastEOLTrimmed, nil
}

func (pp PrettyPrinter) sprintPointer(v any) (string, error) {
	_, wasVisited := pp.visitedPtrs[v]
	if wasVisited {
		return fmt.Sprintf("@ %p", v), nil
	}

	pp.visitedPtrs[v] = true

	vValue := reflect.ValueOf(v)

	if !vValue.IsNil() {
		return pp.sprintStruct(vValue.Elem().Interface())
	} else {
		return fmt.Sprintf("@ %p", v), nil
	}
}

func (pp PrettyPrinter) SprintPretty(v any) (string, error) {
	vKind := reflect.TypeOf(v).Kind()

	switch vKind {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32,
		reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
		return pp.sprintPrimitive(v)
	case reflect.Slice:
		return pp.sprintSlice(v)
	case reflect.Struct:
		return pp.sprintStruct(v)
	case reflect.Pointer:
		return pp.sprintPointer(v)
	default:
		return "", fmt.Errorf("unsupported kind: %s", vKind)
	}
}

func sprintIndent(s string, prefixSymbol string, width uint) string {
	s = strings.TrimRight(s, "\n")
	lines := strings.Split(s, "\n")
	prefix := strings.Repeat(prefixSymbol, int(width))

	for i, l := range lines {
		lines[i] = prefix + l
	}

	return strings.Join(lines, "\n")
}
