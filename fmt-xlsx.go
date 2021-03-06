package main

import (
	"fmt"
	"github.com/tealeg/xlsx"
	"strings"
)

const (
	xlsxSheetMeta       = "Meta"
	xlsxMetaAuthor      = "author"
	xlsxMetaName        = "name"
	xlsxMetaVersion     = "version"
	xlsxDefaultFontName = "Verdana"
	xlsxDefaultFontSize = 10

	headerNullable = "nullable"
	headerNotNull  = "not null"
)

type Xlsx struct {
	metaSheet        *xlsx.Sheet
	sheetsByGroup    map[string]*xlsx.Sheet
	UseNotNullColumn bool
}

func (x *Xlsx) FromFile(filename string) error {
	xlFile, err := xlsx.OpenFile(filename)
	if err != nil {
		return err
	}

	x.sheetsByGroup = make(map[string]*xlsx.Sheet)

	for _, sheet := range xlFile.Sheets {
		sheetName := sheet.Name
		if sheetName == xlsxSheetMeta {
			x.metaSheet = sheet
		} else {
			x.sheetsByGroup[sheet.Name] = sheet
		}
	}
	return nil
}

func (x *Xlsx) ToSchema() (*Schema, error) {
	author := ""
	name := ""
	version := ""
	tables := make([]*Table, 0)

	if x.metaSheet != nil {
		keyValues := x.readMetaSheet()
		author = keyValues[xlsxMetaAuthor]
		name = keyValues[xlsxMetaName]
		version = keyValues[xlsxMetaVersion]
	}

	for groupName, sheet := range x.sheetsByGroup {
		groupTables, err := x.readGroupSheet(groupName, sheet)
		if err != nil {
			return nil, err
		}
		tables = append(tables, groupTables...)
	}

	return &Schema{
		Author:  author,
		Name:    name,
		Version: version,
		Tables:  tables,
	}, nil
}

func (x *Xlsx) readMetaSheet() map[string]string {
	result := map[string]string{}

	for _, row := range x.metaSheet.Rows {
		if keyCell := x.getCell(row, 0); keyCell != nil {
			key := keyCell.Value
			valueCell := x.getCell(row, 1)
			if valueCell == nil {
				result[key] = ""
			} else {
				result[key] = valueCell.Value
			}
		}
	}
	return result
}

func (x *Xlsx) readGroupSheet(groupName string, sheet *xlsx.Sheet) ([]*Table, error) {
	tables := make([]*Table, 0)

	var lastTable *Table
	tableFinished := true
	useNotNullColumn := false
	for i, row := range sheet.Rows {
		// skip header row
		if i == 0 {
			if strings.TrimSpace(x.getCellValue(row, 4)) == headerNotNull {
				useNotNullColumn = true
			}
			continue
		}

		tableName := strings.TrimSpace(x.getCellValue(row, 0))
		columnName := strings.TrimSpace(x.getCellValue(row, 1))

		// finish table if
		// - column is empty
		if columnName == "" && !tableFinished {
			tables = append(tables, lastTable)
			tableFinished = true
			continue
		}

		typeValue := strings.TrimSpace(x.getCellValue(row, 2))
		keyValue := strings.TrimSpace(x.getCellValue(row, 3))
		nullableValue := strings.TrimSpace(x.getCellValue(row, 4))
		attrValue := strings.TrimSpace(x.getCellValue(row, 5))
		description := strings.TrimSpace(x.getCellValue(row, 6))

		// create new table
		if tableFinished {
			if tableName != "" {
				lastTable = &Table{
					Name:        tableName,
					Columns:     make([]*Column, 0),
					Description: description,
					Group:       groupName,
					ClassName:   typeValue,
				}
				tableFinished = false
			}
			continue
		}
		if lastTable == nil {
			continue
		}

		// column type
		colType, colSize, colScale := ParseType(typeValue)

		// add column
		defaultValue := ""
		attrSet := NewStringSet()
		for _, attr := range strings.Split(attrValue, ",") {
			attr = strings.TrimSpace(attr)

			if strings.HasPrefix(attr, "default") {
				tokens := strings.SplitN(attr, ":", 2)
				if len(tokens) == 2 {
					defaultValue = x.fixDefaultValue(colType, tokens[1])
					continue
				}
			}

			attrSet.Add(strings.ToLower(attr))
		}

		// reference
		var ref *Reference
		if tableName != "" {
			tokens := strings.Split(tableName, ".")
			if len(tokens) == 2 {
				ref = &Reference{
					Table:  tokens[0],
					Column: tokens[1],
				}
			}
		}

		lastTable.AddColumn(&Column{
			Name:            columnName,
			Type:            colType,
			Description:     description,
			Size:            colSize,
			Scale:           colScale,
			Nullable:        TernaryBool(useNotNullColumn, nullableValue == "", nullableValue != ""),
			PrimaryKey:      keyValue == "P",
			UniqueKey:       keyValue == "U",
			AutoIncremental: attrSet.ContainsAny([]string{"ai", "autoinc", "auto_inc", "auto_incremental"}),
			DefaultValue:    defaultValue,
			Ref:             ref,
		})
	}

	if !tableFinished && lastTable != nil {
		tables = append(tables, lastTable)
	}

	return tables, nil
}

func (x *Xlsx) fixDefaultValue(colType string, defaultValue string) string {
	if IsBooleanType(colType) {
		return TernaryString(defaultValue == "true" || defaultValue == "1", "true", "false")
	}
	return defaultValue
}

func (x *Xlsx) ToFile(schema *Schema, filename string) error {
	file := xlsx.NewFile()
	metaSheet, err := file.AddSheet(xlsxSheetMeta)
	if err != nil {
		return err
	}

	if err = x.fillMetaSheet(metaSheet, schema); err != nil {
		return err
	}

	for _, group := range schema.Groups() {
		groupName := group
		if groupName == "" {
			groupName = "Common"
		}
		sheet, err := file.AddSheet(groupName)
		if err != nil {
			return err
		}

		sheet.SheetViews = []xlsx.SheetView{
			{
				Pane: &xlsx.Pane{
					XSplit:      2,
					YSplit:      1,
					TopLeftCell: "C2",
					ActivePane:  "bottomRight",
					State:       "frozen",
				},
			},
		}

		if err = x.fillGroupSheet(sheet, schema, group); err != nil {
			return err
		}
	}

	return file.Save(filename)
}

func (x *Xlsx) fillMetaSheet(sheet *xlsx.Sheet, schema *Schema) error {
	_ = sheet.SetColWidth(0, 0, 10.5)
	_ = sheet.SetColWidth(1, 1, 10.5)
	font := x.defaultFont()
	style := x.newStyle(nil, nil, nil, font)

	row := sheet.AddRow()
	x.addCells(row, []string{xlsxMetaAuthor, schema.Author}, style)
	row = sheet.AddRow()
	x.addCells(row, []string{xlsxMetaName, schema.Name}, style)
	row = sheet.AddRow()
	x.addCells(row, []string{xlsxMetaVersion, schema.Version}, style)
	return nil
}

func (x *Xlsx) newBorder(thickness, color string) *xlsx.Border {
	border := xlsx.NewBorder(thickness, thickness, thickness, thickness)
	if color != "" {
		border.LeftColor = color
		border.RightColor = color
		border.TopColor = color
		border.BottomColor = color
	}
	return border
}

func (x *Xlsx) newSolidFill(color string) *xlsx.Fill {
	return xlsx.NewFill("solid", color, color)
}

func (x *Xlsx) newAlignment(horizontal, vertical string) *xlsx.Alignment {
	return &xlsx.Alignment{
		Horizontal: horizontal,
		Vertical:   vertical,
	}
}

func (x *Xlsx) defaultFont() *xlsx.Font {
	return xlsx.NewFont(xlsxDefaultFontSize, xlsxDefaultFontName)
}

func (x *Xlsx) newStyle(
	fill *xlsx.Fill,
	border *xlsx.Border,
	alignment *xlsx.Alignment,
	font *xlsx.Font,
) *xlsx.Style {
	style := xlsx.NewStyle()
	if fill != nil {
		style.ApplyFill = true
		style.Fill = *fill
	}
	if border != nil {
		style.ApplyBorder = true
		style.Border = *border
	}
	if alignment != nil {
		style.ApplyAlignment = true
		style.Alignment = *alignment
	}
	if font != nil {
		style.ApplyFont = true
		style.Font = *font
	}

	return style
}

func (x *Xlsx) fillGroupSheet(sheet *xlsx.Sheet, schema *Schema, group string) error {
	_ = sheet.SetColWidth(0, 0, 18)
	_ = sheet.SetColWidth(1, 1, 13.5)
	_ = sheet.SetColWidth(2, 2, 9.5)
	_ = sheet.SetColWidth(3, 3, 4.0)
	_ = sheet.SetColWidth(4, 4, TernaryFloat64(x.UseNotNullColumn, 6.0, 4.0))
	_ = sheet.SetColWidth(5, 5, 9.5)
	_ = sheet.SetColWidth(6, 6, 50)

	// alignment
	leftAlignment := x.newAlignment("default", "center")
	centerAlignment := x.newAlignment("center", "center")

	// border
	border := x.newBorder("thin", "")
	lightBorder := x.newBorder("thin", "00B2B2B2")

	// font
	boldFont := x.defaultFont()
	boldFont.Bold = true
	normalFont := x.defaultFont()
	refFont := xlsx.NewFont(8, xlsxDefaultFontName)
	refFont.Italic = true

	headerStyle := x.newStyle(x.newSolidFill("00CCFFCC"), border, centerAlignment, boldFont)
	tableStyle := x.newStyle(x.newSolidFill("00CCFFFF"), border, centerAlignment, boldFont)
	tableDescStyle := x.newStyle(x.newSolidFill("00FFFBCC"), lightBorder, leftAlignment, normalFont)
	normalStyle := x.newStyle(nil, lightBorder, leftAlignment, normalFont)
	boolStyle := x.newStyle(nil, lightBorder, centerAlignment, normalFont)
	referenceStyle := x.newStyle(nil, lightBorder, centerAlignment, refFont)

	// Header
	row := sheet.AddRow()
	nullHeaderText := TernaryString(x.UseNotNullColumn, headerNotNull, headerNullable)

	x.addCells(row, []string{
		"Table/Reference",
		"Column",
		"Type",
		"Key",
		nullHeaderText,
		"Attributes",
		"Description",
	}, headerStyle)

	tableCount := len(schema.Tables)
	for i, table := range schema.Tables {
		if table.Group != group {
			continue
		}

		// Table
		row = sheet.AddRow()
		x.addCell(row, table.Name, tableStyle)
		x.addCells(row, []string{"", "", "", "", ""}, nil)
		x.addCell(row, strings.TrimSpace(table.Description), tableDescStyle)

		// Columns
		for _, column := range table.Columns {
			row = sheet.AddRow()

			// table/Reference
			if ref := x.getColumnReference(column); ref != "" {
				x.addCell(row, ref, referenceStyle)
			} else {
				x.addCell(row, "", nil)
			}
			// column
			x.addCell(row, column.Name, normalStyle)
			// type
			x.addCell(row, x.formatType(column), normalStyle)
			// key
			x.addCell(row, BoolToString(column.PrimaryKey, "P", BoolToString(column.UniqueKey, "U", "")), boolStyle)
			// nullable
			if x.UseNotNullColumn {
				x.addCell(row, BoolToString(!column.Nullable, "O", ""), boolStyle)
			} else {
				x.addCell(row, BoolToString(column.Nullable, "O", ""), boolStyle)
			}
			// attributes
			x.addCell(row, strings.Join(x.getColumnAttributes(column), ", "), normalStyle)
			// description
			x.addCell(row, strings.TrimSpace(column.Description), normalStyle)
		}

		// add empty row
		if i < tableCount-1 {
			sheet.AddRow()
		}
	}

	return nil
}

func (x *Xlsx) getCell(row *xlsx.Row, colIdx int) *xlsx.Cell {
	colCount := len(row.Cells)
	if colIdx < colCount {
		return row.Cells[colIdx]
	}
	return nil
}

func (x *Xlsx) getCellValue(row *xlsx.Row, colIdx int) string {
	cell := x.getCell(row, colIdx)
	if cell == nil {
		return ""
	} else {
		return cell.Value
	}
}

func (x *Xlsx) addCell(row *xlsx.Row, value string, style *xlsx.Style) *xlsx.Cell {
	cell := row.AddCell()
	cell.Value = value
	if style != nil {
		cell.SetStyle(style)
	}
	return cell
}

func (x *Xlsx) addCells(row *xlsx.Row, values []string, style *xlsx.Style) {
	for _, value := range values {
		x.addCell(row, value, style)
	}
}

func (x *Xlsx) getColumnAttributes(column *Column) []string {
	attrs := make([]string, 0)

	if column.AutoIncremental {
		attrs = append(attrs, "autoInc")
	}
	if column.DefaultValue != "" {
		attrs = append(attrs, "default:"+column.DefaultValue)
	}

	return attrs
}

func (x *Xlsx) getColumnReference(column *Column) string {
	ref := column.Ref
	if ref == nil {
		return ""
	}
	return fmt.Sprintf("%s.%s", ref.Table, ref.Column)
}

func (x *Xlsx) formatType(column *Column) string {
	if column.Size == 0 {
		return column.Type
	}
	if column.Scale == 0 {
		return fmt.Sprintf("%s(%d)", column.Type, column.Size)
	}
	return fmt.Sprintf("%s(%d,%d)", column.Type, column.Size, column.Scale)
}
