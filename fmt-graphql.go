package main

import (
	"fmt"
	"github.com/gertd/go-pluralize"
	"github.com/iancoleman/strcase"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

type GraphqlClass struct {
	table    *Table
	Name     string
	Fields   []*GraphqlField
	PKFields []*GraphqlField
}

type GraphqlField struct {
	Column *Column
	Name   string
	Type   string
}

type Graphql struct {
}

func NewGraphqlClass(
	table *Table,
	output *Output,
	prefixMapper *PrefixMapper,
) *GraphqlClass {
	className := table.ClassName
	if className == "" {
		tableName := table.Name
		prefixes := output.GetSlice(FlagRemovePrefix)
		for _, prefix := range prefixes {
			tableName = strings.TrimPrefix(tableName, prefix)
		}
		className = strcase.ToCamel(tableName)

		if prefix := prefixMapper.GetPrefix(table.Group); prefix != "" {
			className = prefix + className
		}
	}

	fields := make([]*GraphqlField, 0)
	pkFields := make([]*GraphqlField, 0)
	for _, column := range table.Columns {
		field := NewGraphqlField(column)
		fields = append(fields, field)

		if column.PrimaryKey {
			pkFields = append(pkFields, field)
		}
	}

	if len(pkFields) == 1 {
		pkFields[0].Type = "ID!"
	}

	return &GraphqlClass{
		table:  table,
		Name:   className,
		Fields: fields,
	}
}

func NewGraphqlField(column *Column) *GraphqlField {
	var fieldType string
	nullable := column.Nullable
	columnType := strings.ToLower(column.Type)
	switch columnType {
	case ColTypeDateTime:
		fallthrough
	case ColTypeDate:
		fallthrough
	case ColTypeTime:
		fallthrough
	case ColTypeString:
		fallthrough
	case ColTypeText:
		fieldType = "String"
	case ColTypeBoolean:
		fieldType = "Boolean"
	case ColTypeLong:
		fallthrough
	case ColTypeInt:
		fieldType = "Int"
	case ColTypeDecimal:
		fallthrough
	case ColTypeFloat:
		fallthrough
	case ColTypeDouble:
		fieldType = "Float"
	default:
		log.Printf("unknown column type: '%s', column: %s", column.Type, column.Name)
		fieldType = "String"
	}
	if !nullable {
		fieldType = fieldType + "!"
	}

	return &GraphqlField{
		Column: column,
		Name:   strcase.ToLowerCamel(column.Name),
		Type:   fieldType,
	}
}

func (g *Graphql) Generate(
	schema *Schema,
	output *Output,
	tableFilterFn TableFilterFn,
	prefixMapper *PrefixMapper,
) error {
	// Create directory
	if err := os.MkdirAll(output.FilePath, 0777); err != nil {
		return err
	}
	log.Printf("[MKDIR] %s", output.FilePath)

	indent := "  "
	contents := make([]string, 0)
	appendLine := func(indentLevel int, line string) {
		contents = append(contents,
			strings.Repeat(indent, indentLevel)+line,
		)
	}

	appendLine(0, `schema {
    query: Query
}
`)

	classes := make([]*GraphqlClass, 0)

	client := pluralize.NewClient()

	appendLine(0, "type Query {")
	for _, table := range schema.Tables {
		// filter table
		if tableFilterFn != nil && !tableFilterFn(table) {
			continue
		}

		class := NewGraphqlClass(table, output, prefixMapper)
		classes = append(classes, class)

		lowerClassName := strcase.ToLowerCamel(class.Name)
		appendLine(1, fmt.Sprintf("%s: [%s]", client.Plural(lowerClassName), class.Name))
	}
	appendLine(0, "}")
	appendLine(0, "")

	for _, class := range classes {
		appendLine(0, fmt.Sprintf("type %s {", class.Name))

		for _, field := range class.Fields {
			appendLine(1, fmt.Sprintf("%s: %s", field.Name, field.Type))
		}
		appendLine(0, "}")
		appendLine(0, "")
	}

	// Write file
	outputFile := path.Join(output.FilePath,
		fmt.Sprintf("%s-%s.graphqls", schema.Name, schema.Version))

	if err := ioutil.WriteFile(outputFile, []byte(strings.Join(contents, "\n")), 0644); err != nil {
		return err
	}
	log.Printf("[WRITE] %s", outputFile)

	return nil
}
