package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
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

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal">
    <w:name w:val="Normal"/>
    <w:qFormat/>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading1">
    <w:name w:val="heading 1"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:before="240" w:after="120"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="32"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading2">
    <w:name w:val="heading 2"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:before="180" w:after="90"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="28"/></w:rPr>
  </w:style>
  <w:style w:type="paragraph" w:styleId="Heading3">
    <w:name w:val="heading 3"/>
    <w:basedOn w:val="Normal"/>
    <w:qFormat/>
    <w:pPr><w:spacing w:before="120" w:after="60"/></w:pPr>
    <w:rPr><w:b/><w:sz w:val="24"/></w:rPr>
  </w:style>
  <w:style w:type="table" w:styleId="TableGrid">
    <w:name w:val="Table Grid"/>
    <w:tblPr>
      <w:tblBorders>
        <w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:left w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:right w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        <w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>
      </w:tblBorders>
    </w:tblPr>
  </w:style>
</w:styles>`

func buildDocumentXML(report Report, sections []ReportSection) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	writeHeading(&b, report.Name, 1)
	writeParagraph(&b, report.Topic)
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
			writeParagraph(&b, para)
		}
		for _, table := range section.Tables {
			writeTable(&b, table)
		}
	}
	b.WriteString(`<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr>`)
	b.WriteString(`</w:body></w:document>`)
	return b.String()
}

func writeHeading(b *strings.Builder, text string, level int) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	styleID := fmt.Sprintf("Heading%d", level)
	b.WriteString(`<w:p>`)
	b.WriteString(`<w:pPr><w:pStyle w:val="`)
	b.WriteString(styleID)
	b.WriteString(`"/></w:pPr>`)
	b.WriteString(`<w:r><w:t>`)
	xml.EscapeText(b, []byte(text))
	b.WriteString(`</w:t></w:r></w:p>`)
}

func writeParagraph(b *strings.Builder, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	b.WriteString(`<w:p><w:r><w:t>`)
	xml.EscapeText(b, []byte(text))
	b.WriteString(`</w:t></w:r></w:p>`)
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

	b.WriteString(`<w:tbl>`)
	b.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="5000" w:type="pct"/></w:tblPr>`)

	if len(headers) > 0 {
		b.WriteString(`<w:tr>`)
		for _, h := range headers {
			b.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr>`)
			b.WriteString(`<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>`)
			xml.EscapeText(b, []byte(h))
			b.WriteString(`</w:t></w:r></w:p></w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}

	for _, row := range rows {
		b.WriteString(`<w:tr>`)
		for _, cell := range row {
			b.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr>`)
			b.WriteString(`<w:p><w:r><w:t>`)
			xml.EscapeText(b, []byte(cell))
			b.WriteString(`</w:t></w:r></w:p></w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}

	b.WriteString(`</w:tbl>`)
	b.WriteString(`<w:p/>`)

	if strings.TrimSpace(footnote) != "" {
		b.WriteString(`<w:p><w:r><w:rPr><w:sz w:val="18"/><w:color w:val="595959"/></w:rPr><w:t>`)
		xml.EscapeText(b, []byte(strings.TrimSpace(footnote)))
		b.WriteString(`</w:t></w:r></w:p>`)
	}
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
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
