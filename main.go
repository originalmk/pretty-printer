package main

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type Person struct {
	Name      string     `pretty:"sem=title,ord=1"`
	Surname   string     `pretty:"ord=2"`
	Age       uint       `pretty:"ord=3"`
	Abilities []Ability  `pretty:"ord=4"`
	Computer  Computer   `pretty:"ord=6"`
	Servers   []Computer `pretty:"ord=5"`
}

type Computer struct {
	CPU string
}

type Ability struct {
	Name  string `pretty:"sem=title"`
	Level uint
}

func getPerson() Person {
	return Person{
		Name:      "John",
		Surname:   "Doe",
		Age:       31,
		Abilities: getAbilities(),
		Computer: Computer{
			CPU: "Ryzen 7700",
		},
		Servers: []Computer{
			{CPU: "Cortex A76"},
		},
	}
}

func getAbilities() []Ability {
	return []Ability{
		{"C Programmer", 5},
		{"Go Programmer", 4},
	}
}

type ParsedTags struct {
	isTitle    bool
	orderIndex uint
}

func parseTags(rawTags map[string]string) ParsedTags {
	result := ParsedTags{}

	semTag, ok := rawTags["sem"]
	if ok {
		result.isTitle = (semTag == "title")
	}

	ordTag, ok := rawTags["ord"]
	if ok {
		orderIndex, _ := strconv.Atoi(ordTag)
		result.orderIndex = uint(orderIndex)
	}

	return result
}

func prettyTags(field reflect.StructField) (ParsedTags, error) {
	result := make(map[string]string)
	prettyTag := field.Tag.Get("pretty")

	if prettyTag != "" {
		for tagElement := range strings.SplitSeq(prettyTag, ",") {
			elementSplit := strings.Split(tagElement, "=")

			if len(elementSplit) != 2 {
				return ParsedTags{}, errors.New("invalid tag")
			}

			elementKey := elementSplit[0]
			elementValue := elementSplit[1]

			result[elementKey] = elementValue
		}
	}

	return parseTags(result), nil
}

type AnnotatedField struct {
	Name  string
	Value reflect.Value
	Tags  ParsedTags
}

type AnnotatedFields = map[string]AnnotatedField

func annotateFields(v any) (AnnotatedFields, error) {
	result := make(AnnotatedFields)
	vType := reflect.TypeOf(v)
	vFields := reflect.VisibleFields(vType)
	vValue := reflect.ValueOf(v)

	for _, field := range vFields {
		parsedTags, err := prettyTags(field)
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

type AnnotatedStruct struct {
	HasTitleField   bool
	TitleField      AnnotatedField
	AnnotatedFields AnnotatedFields
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

type printFunction func(any, SprintOptions) (string, error)

var printHandlers map[reflect.Kind]printFunction

func init() {
	printHandlers = map[reflect.Kind]printFunction{
		// Primitives
		reflect.Bool:       sprintPrimitive,
		reflect.Int:        sprintPrimitive,
		reflect.Int8:       sprintPrimitive,
		reflect.Int16:      sprintPrimitive,
		reflect.Int32:      sprintPrimitive,
		reflect.Int64:      sprintPrimitive,
		reflect.Uint:       sprintPrimitive,
		reflect.Uint8:      sprintPrimitive,
		reflect.Uint16:     sprintPrimitive,
		reflect.Uint32:     sprintPrimitive,
		reflect.Uint64:     sprintPrimitive,
		reflect.Float32:    sprintPrimitive,
		reflect.Float64:    sprintPrimitive,
		reflect.Complex64:  sprintPrimitive,
		reflect.Complex128: sprintPrimitive,
		reflect.String:     sprintPrimitive,
		// Complex types
		reflect.Slice:  sprintSlice,
		reflect.Struct: sprintStruct,
	}
}

func sprintPrimitive(v any, options SprintOptions) (string, error) {
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

func sprintSlice(v any, options SprintOptions) (string, error) {
	if reflect.TypeOf(v).Kind() != reflect.Slice {
		return "", errors.New("kind is not slice")
	}

	var resultBuilder strings.Builder

	if !options.SkipHeader {
		resultBuilder.WriteString("List")
		resultBuilder.WriteString("\n")
	}

	vValue := reflect.ValueOf(v)

	var listItemsBuilder strings.Builder

	for i := 0; i < vValue.Len(); i++ {
		s, err := sprintPretty(vValue.Index(i).Interface(), SprintOptionsDefault())
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

	listItemsSprint := listItemsBuilder.String()
	listItemsIndented := sprintIndent(listItemsSprint, " ", 4)
	resultBuilder.WriteString(listItemsIndented)

	lastEOLTrimmed := strings.TrimRight(resultBuilder.String(), "\n")

	return lastEOLTrimmed, nil
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

func sprintStruct(v any, options SprintOptions) (string, error) {
	if reflect.TypeOf(v).Kind() != reflect.Struct {
		return "", errors.New("kind is not struct")
	}

	annotated, err := annotateStruct(v)
	if err != nil {
		return "", err
	}

	var resultBuilder strings.Builder

	if annotated.HasTitleField {
		prettyPrimitive, err := sprintPrimitive(annotated.TitleField.Value.Interface(), SprintOptionsDefault())
		if err != nil {
			return "", err
		}

		resultBuilder.WriteString(prettyPrimitive)
		resultBuilder.WriteString("\n")
	} else if !options.SkipHeader {
		resultBuilder.WriteString(reflect.TypeOf(v).Name())
		resultBuilder.WriteString("\n")
	}

	fieldKeys := slices.Collect(maps.Keys(annotated.AnnotatedFields))
	slices.SortFunc(fieldKeys, func(a string, b string) int {
		aVal := annotated.AnnotatedFields[a]
		bVal := annotated.AnnotatedFields[b]

		return int(aVal.Tags.orderIndex - bVal.Tags.orderIndex)
	})

	for _, fieldKey := range fieldKeys {
		field := annotated.AnnotatedFields[fieldKey]

		if annotated.TitleField.Name == field.Name {
			continue
		}

		var fieldBuilder strings.Builder

		innerOptions := SprintOptionsDefault()
		innerOptions.SkipHeader = true
		innerSprint, err := sprintPretty(field.Value.Interface(), innerOptions)
		if err != nil {
			return "", err
		}

		innerKind := field.Value.Kind()

		fieldBuilder.WriteString(field.Name)

		if len(strings.Split(innerSprint, "\n")) > 1 || innerKind == reflect.Struct || innerKind == reflect.Slice {
			fieldBuilder.WriteString(" >>>\n")
		} else {
			fieldBuilder.WriteString(" = ")
		}

		fieldBuilder.WriteString(innerSprint)

		fieldSprint := fieldBuilder.String()
		indentedSprint := sprintIndent(fieldSprint, " ", 4)

		resultBuilder.WriteString(indentedSprint)
		resultBuilder.WriteString("\n")
	}

	lastEOLTrimmed := strings.TrimRight(resultBuilder.String(), "\n")

	return lastEOLTrimmed, nil
}

func sprintPretty(v any, options SprintOptions) (string, error) {
	vKind := reflect.TypeOf(v).Kind()
	vSprintHandler, ok := printHandlers[vKind]
	if !ok {
		return "", fmt.Errorf("unsupported kind: %s", vKind)
	}

	return vSprintHandler(v, options)
}

type SprintOptions struct {
	SkipHeader bool
}

func SprintOptionsDefault() SprintOptions {
	return SprintOptions{
		SkipHeader: false,
	}
}

func main() {
	structText, err := sprintStruct(getPerson(), SprintOptionsDefault())
	if err != nil {
		panic(err)
	}

	fmt.Println(structText)
}
