package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type SimpleDOCXGenerator struct{}

func NewSimpleDOCXGenerator() *SimpleDOCXGenerator {
	return &SimpleDOCXGenerator{}
}

func (g *SimpleDOCXGenerator) GenerateDOCX(ctx context.Context, report Report, sections []ReportSection) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)
	files := map[string]string{
		"[Content_Types].xml":          contentTypesXML,
		"_rels/.rels":                  packageRelsXML,
		"word/_rels/document.xml.rels": documentRelsXML,
		"word/document.xml":            buildDocumentXML(report, sections),
		"word/styles.xml":              stylesXML,
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		writer, err := zipWriter.Create(name)
		if err != nil {
			_ = zipWriter.Close()
			return nil, fmt.Errorf("create docx part %s: %w", name, err)
		}
		if _, err := writer.Write([]byte(files[name])); err != nil {
			_ = zipWriter.Close()
			return nil, fmt.Errorf("write docx part %s: %w", name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close docx package: %w", err)
	}
	return buffer.Bytes(), nil
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`

const packageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

const documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

// textWidth is the usable line width in twentieths-of-a-point (twips).
// Page width 11906 − left margin 1800 − right margin 1800 = 8306.
const textWidth = 8306

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
          xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml">
  <w:docDefaults>
    <w:rPrDefault>
      <w:rPr>
        <w:rFonts w:ascii="Times New Roman" w:eastAsia="宋体" w:hAnsi="Times New Roman" w:cs="Times New Roman"/>
        <w:sz w:val="24"/>
        <w:szCs w:val="24"/>
      </w:rPr>
    </w:rPrDefault>
    <w:pPrDefault>
      <w:pPr>
        <w:spacing w:line="360" w:lineRule="auto" w:after="120"/>
      </w:pPr>
    </w:pPrDefault>
  </w:docDefaults>

  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:qFormat/>
  </w:style>

  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:before="360" w:after="180" w:line="360" w:lineRule="auto"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Arial" w:eastAsia="黑体" w:hAnsi="Arial"/>
      <w:b/><w:sz w:val="32"/><w:szCs w:val="32"/>
    </w:rPr>
  </w:style>

  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:before="240" w:after="120" w:line="360" w:lineRule="auto"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Arial" w:eastAsia="黑体" w:hAnsi="Arial"/>
      <w:b/><w:sz w:val="28"/><w:szCs w:val="28"/>
    </w:rPr>
  </w:style>

  <w:style w:type="paragraph" w:styleId="Heading3">
    <w:name w:val="heading 3"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:before="180" w:after="60" w:line="360" w:lineRule="auto"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Arial" w:eastAsia="黑体" w:hAnsi="Arial"/>
      <w:b/><w:sz w:val="24"/><w:szCs w:val="24"/>
    </w:rPr>
  </w:style>

  <w:style w:type="paragraph" w:styleId="ListParagraph">
    <w:name w:val="List Paragraph"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr>
      <w:ind w:left="420" w:hanging="420"/>
      <w:spacing w:line="360" w:lineRule="auto" w:after="60"/>
    </w:pPr>
  </w:style>

  <w:style w:type="paragraph" w:styleId="TableFootnote">
    <w:name w:val="Table Footnote"/>
    <w:basedOn w:val="Normal"/>
    <w:pPr><w:spacing w:before="60" w:after="120" w:line="280" w:lineRule="auto"/></w:pPr>
    <w:rPr>
      <w:rFonts w:ascii="Times New Roman" w:eastAsia="宋体" w:hAnsi="Times New Roman"/>
      <w:sz w:val="18"/><w:szCs w:val="18"/>
      <w:color w:val="595959"/>
    </w:rPr>
  </w:style>

  <w:style w:type="table" w:styleId="TableGrid">
    <w:name w:val="Table Grid"/>
    <w:tblPr>
      <w:tblBorders>
        <w:top    w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:left   w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:right  w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      </w:tblBorders>
      <w:tblCellMar>
        <w:top    w:w="60" w:type="dxa"/>
        <w:left   w:w="108" w:type="dxa"/>
        <w:bottom w:w="60" w:type="dxa"/>
        <w:right  w:w="108" w:type="dxa"/>
      </w:tblCellMar>
    </w:tblPr>
    <w:rPr>
      <w:rFonts w:ascii="Times New Roman" w:eastAsia="宋体" w:hAnsi="Times New Roman"/>
      <w:sz w:val="22"/><w:szCs w:val="22"/>
    </w:rPr>
  </w:style>
</w:styles>`

func buildDocumentXML(report Report, sections []ReportSection) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)

	// Cover: centered title + subtitle
	writeTitlePage(&b, report.Name, report.Topic)

	ordered := append([]ReportSection(nil), sections...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SortOrder == ordered[j].SortOrder {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].SortOrder < ordered[j].SortOrder
	})

	for _, section := range ordered {
		title := strings.TrimSpace(section.Title)
		if title != "" {
			if section.Numbering != "" {
				title = section.Numbering + " " + title
			}
			level := section.Level
			if level < 1 {
				level = 1
			}
			if level > 3 {
				level = 3
			}
			writeHeading(&b, title, level)
		}
		for _, para := range splitParagraphs(section.Content) {
			writeBodyParagraph(&b, para)
		}
		for _, table := range section.Tables {
			writeTable(&b, table)
		}
	}

	b.WriteString(`<w:sectPr>`)
	b.WriteString(`<w:pgSz w:w="11906" w:h="16838"/>`)
	b.WriteString(`<w:pgMar w:top="1800" w:right="1800" w:bottom="1800" w:left="1800" w:header="720" w:footer="720" w:gutter="0"/>`)
	b.WriteString(`</w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

// writeTitlePage writes a centered title and subtitle followed by a page break.
func writeTitlePage(b *strings.Builder, title, subtitle string) {
	title = strings.TrimSpace(title)
	if title == "" {
		return
	}
	// Centred main title – large bold black text
	b.WriteString(`<w:p><w:pPr><w:jc w:val="center"/><w:spacing w:before="2880" w:after="360"/></w:pPr>`)
	b.WriteString(`<w:r><w:rPr>`)
	b.WriteString(`<w:rFonts w:ascii="Arial" w:eastAsia="黑体" w:hAnsi="Arial"/>`)
	b.WriteString(`<w:b/><w:sz w:val="48"/><w:szCs w:val="48"/>`)
	b.WriteString(`</w:rPr><w:t xml:space="preserve">`)
	xml.EscapeText(b, []byte(title))
	b.WriteString(`</w:t></w:r></w:p>`)

	// Centred subtitle
	if sub := strings.TrimSpace(subtitle); sub != "" {
		b.WriteString(`<w:p><w:pPr><w:jc w:val="center"/><w:spacing w:before="120" w:after="2880"/></w:pPr>`)
		b.WriteString(`<w:r><w:rPr>`)
		b.WriteString(`<w:rFonts w:ascii="Times New Roman" w:eastAsia="宋体" w:hAnsi="Times New Roman"/>`)
		b.WriteString(`<w:sz w:val="28"/><w:szCs w:val="28"/>`)
		b.WriteString(`</w:rPr><w:t xml:space="preserve">`)
		xml.EscapeText(b, []byte(sub))
		b.WriteString(`</w:t></w:r></w:p>`)
	}

	// Page break before content
	b.WriteString(`<w:p><w:r><w:br w:type="page"/></w:r></w:p>`)
}

func writeHeading(b *strings.Builder, text string, level int) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	styleID := fmt.Sprintf("Heading%d", level)
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="`)
	b.WriteString(styleID)
	b.WriteString(`"/></w:pPr>`)
	b.WriteString(`<w:r><w:t xml:space="preserve">`)
	xml.EscapeText(b, []byte(text))
	b.WriteString(`</w:t></w:r></w:p>`)
}

// writeBodyParagraph detects list items and applies ListParagraph style; otherwise Normal.
func writeBodyParagraph(b *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if isListItem(text) {
		b.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListParagraph"/></w:pPr>`)
	} else {
		b.WriteString(`<w:p>`)
	}
	b.WriteString(`<w:r><w:t xml:space="preserve">`)
	xml.EscapeText(b, []byte(text))
	b.WriteString(`</w:t></w:r></w:p>`)
}

// isListItem returns true when a line looks like a numbered or bulleted list entry.
func isListItem(s string) bool {
	if strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "• ") || strings.HasPrefix(s, "· ") {
		return true
	}
	// Arabic numeral: "1. " / "1、" / "1）"
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i < len(s) {
		rest := s[i:]
		if strings.HasPrefix(rest, ". ") || strings.HasPrefix(rest, "、") || strings.HasPrefix(rest, "）") {
			return true
		}
	}
	// Chinese numeral followed by 、 (e.g. 一、二、三)
	if len(s) >= 3 {
		r := []rune(s)
		if isChineseOrdinal(r[0]) && string(r[1]) == "、" {
			return true
		}
	}
	return false
}

func isChineseOrdinal(r rune) bool {
	ordinals := "一二三四五六七八九十"
	return strings.ContainsRune(ordinals, r)
}

func writeTable(b *strings.Builder, table map[string]any) {
	if len(table) == 0 {
		return
	}
	headers, _ := toStringSlice(table["headers"])
	rows := toRowSlice(table["rows"])
	footnote, _ := table["footnote"].(string)

	if len(headers) == 0 && len(rows) == 0 {
		return
	}

	// Calculate equal column widths that fill the text area.
	numCols := len(headers)
	if numCols == 0 && len(rows) > 0 {
		numCols = len(rows[0])
	}
	colWidth := textWidth
	if numCols > 0 {
		colWidth = textWidth / numCols
	}
	colWidthStr := fmt.Sprintf("%d", colWidth)
	tableWidthStr := fmt.Sprintf("%d", colWidth*numCols)

	b.WriteString(`<w:tbl>`)
	b.WriteString(`<w:tblPr>`)
	b.WriteString(`<w:tblStyle w:val="TableGrid"/>`)
	b.WriteString(`<w:tblW w:w="` + tableWidthStr + `" w:type="dxa"/>`)
	b.WriteString(`<w:jc w:val="center"/>`)
	b.WriteString(`</w:tblPr>`)

	// Column definitions
	b.WriteString(`<w:tblGrid>`)
	for i := 0; i < numCols; i++ {
		b.WriteString(`<w:gridCol w:w="` + colWidthStr + `"/>`)
	}
	b.WriteString(`</w:tblGrid>`)

	// Header row
	if len(headers) > 0 {
		b.WriteString(`<w:tr>`)
		b.WriteString(`<w:trPr><w:tblHeader/></w:trPr>`)
		for _, h := range headers {
			writeTableCell(b, h, colWidthStr, true)
		}
		b.WriteString(`</w:tr>`)
	}

	// Data rows
	for _, row := range rows {
		b.WriteString(`<w:tr>`)
		for ci, cell := range row {
			_ = ci
			writeTableCell(b, cell, colWidthStr, false)
		}
		b.WriteString(`</w:tr>`)
	}

	b.WriteString(`</w:tbl>`)
	b.WriteString(`<w:p/>`) // required spacing after table

	if fn := strings.TrimSpace(footnote); fn != "" {
		b.WriteString(`<w:p><w:pPr><w:pStyle w:val="TableFootnote"/></w:pPr>`)
		b.WriteString(`<w:r><w:t xml:space="preserve">`)
		xml.EscapeText(b, []byte(fn))
		b.WriteString(`</w:t></w:r></w:p>`)
	}
}

func writeTableCell(b *strings.Builder, text, colWidth string, header bool) {
	b.WriteString(`<w:tc>`)
	b.WriteString(`<w:tcPr><w:tcW w:w="` + colWidth + `" w:type="dxa"/>`)
	if header {
		b.WriteString(`<w:shd w:val="clear" w:color="auto" w:fill="D9D9D9"/>`)
	}
	b.WriteString(`</w:tcPr>`)
	b.WriteString(`<w:p><w:pPr>`)
	if header {
		b.WriteString(`<w:jc w:val="center"/>`)
	}
	b.WriteString(`</w:pPr>`)
	b.WriteString(`<w:r>`)
	if header {
		b.WriteString(`<w:rPr><w:b/></w:rPr>`)
	}
	b.WriteString(`<w:t xml:space="preserve">`)
	xml.EscapeText(b, []byte(text))
	b.WriteString(`</w:t></w:r></w:p></w:tc>`)
}

func toStringSlice(v any) ([]string, bool) {
	raw, ok := v.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		result = append(result, fmt.Sprintf("%v", item))
	}
	return result, true
}

func toRowSlice(v any) [][]string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	rows := make([][]string, 0, len(raw))
	for _, rowRaw := range raw {
		cells, ok := rowRaw.([]any)
		if !ok {
			continue
		}
		row := make([]string, 0, len(cells))
		for _, cell := range cells {
			row = append(row, fmt.Sprintf("%v", cell))
		}
		rows = append(rows, row)
	}
	return rows
}

func splitParagraphs(content string) []string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
