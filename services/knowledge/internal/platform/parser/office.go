package parser

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

const (
	maxOfficeXMLEntryBytes = uint64(maxParsedTextBytes)
	maxOfficeXMLTotalBytes = uint64(maxParsedTextBytes * 2)
	maxPPTXOCRImageBytes   = uint64(4 << 20)
	maxPPTXOCRImages       = 16
)

var errArchiveEntryMissing = errors.New("document archive is missing required content")

func parseDOCX(archive *zip.Reader) (service.ParsedDocument, error) {
	reader := newOfficeArchiveReader(archive)
	data, err := reader.readXML("word/document.xml")
	if err != nil {
		return service.ParsedDocument{}, err
	}
	paragraphs, err := extractParagraphText(data)
	if err != nil {
		return service.ParsedDocument{}, err
	}
	content := strings.TrimSpace(strings.Join(paragraphs, "\n\n"))
	if content == "" {
		return service.ParsedDocument{}, fmt.Errorf("document is empty")
	}
	return service.ParsedDocument{
		Content: content,
		Title:   firstNonEmptyLine(content),
	}, nil
}

type ocrRequestContext struct {
	requestID string
	userID    string
}

func parsePPTX(ctx context.Context, archive *zip.Reader, ocr OCRClient, requestContext ocrRequestContext) (service.ParsedDocument, error) {
	reader := newOfficeArchiveReader(archive)
	slideFiles := orderedPresentationSlides(reader)
	if len(slideFiles) == 0 {
		return service.ParsedDocument{}, fmt.Errorf("presentation has no slides")
	}
	sections := make([]string, 0, len(slideFiles))
	title := ""
	for index, file := range slideFiles {
		data, err := reader.readXML(file)
		if err != nil {
			return service.ParsedDocument{}, err
		}
		paragraphs, err := extractParagraphText(data)
		if err != nil {
			return service.ParsedDocument{}, err
		}
		ocrParagraphs, err := extractSlideOCRText(ctx, reader, file, ocr, requestContext)
		if err != nil {
			return service.ParsedDocument{}, err
		}
		slideText := strings.TrimSpace(strings.Join(append(paragraphs, ocrParagraphs...), "\n"))
		if slideText == "" {
			continue
		}
		if title == "" {
			title = firstPresentationTitle(slideText)
		}
		sections = append(sections, fmt.Sprintf("Slide %d\n%s", index+1, slideText))
	}
	content := strings.TrimSpace(strings.Join(sections, "\n\n"))
	if content == "" {
		return service.ParsedDocument{}, fmt.Errorf("document is empty")
	}
	return service.ParsedDocument{Content: content, Title: title}, nil
}

func firstPresentationTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || (strings.HasPrefix(line, "Image ") && strings.HasSuffix(line, " OCR")) {
			continue
		}
		return line
	}
	return firstNonEmptyLine(content)
}

func extractSlideOCRText(ctx context.Context, archive *officeArchiveReader, slideFile string, ocr OCRClient, requestContext ocrRequestContext) ([]string, error) {
	if ocr == nil {
		return nil, nil
	}
	imageTargets := imageTargetsForSlide(archive, slideFile)
	if len(imageTargets) == 0 {
		return nil, nil
	}
	if len(imageTargets) > archive.remainingOCRImages() {
		return nil, fmt.Errorf("presentation contains too many images for OCR")
	}
	paragraphs := make([]string, 0, len(imageTargets))
	for index, target := range imageTargets {
		data, err := archive.readOCRImage(target)
		if err != nil {
			if errors.Is(err, errArchiveEntryMissing) {
				continue
			}
			return nil, err
		}
		result, err := ocr.ExtractText(ctx, OCRRequest{
			DocumentName: target,
			ContentType:  imageContentType(target, data),
			Data:         data,
			SizeBytes:    int64(len(data)),
			RequestID:    requestContext.requestID,
			UserID:       requestContext.userID,
		})
		if err != nil {
			return nil, service.DependencyError("document OCR failed", err)
		}
		text := strings.TrimSpace(result.Text)
		if text != "" {
			paragraphs = append(paragraphs, fmt.Sprintf("Image %d OCR\n%s", index+1, text))
		}
	}
	return paragraphs, nil
}

func parseXLSX(archive *zip.Reader) (service.ParsedDocument, error) {
	reader := newOfficeArchiveReader(archive)
	sheetFiles := orderedWorkbookSheets(reader)
	if len(sheetFiles) == 0 {
		return service.ParsedDocument{}, fmt.Errorf("spreadsheet has no worksheets")
	}
	sharedStrings, err := parseSharedStrings(reader)
	if err != nil {
		return service.ParsedDocument{}, err
	}
	sections := make([]string, 0, len(sheetFiles))
	for index, file := range sheetFiles {
		data, err := reader.readXML(file)
		if err != nil {
			return service.ParsedDocument{}, err
		}
		rows, err := extractWorksheetRows(data, sharedStrings)
		if err != nil {
			return service.ParsedDocument{}, err
		}
		if len(rows) == 0 {
			continue
		}
		section := append([]string{fmt.Sprintf("Sheet %d", index+1)}, rows...)
		sections = append(sections, strings.Join(section, "\n"))
	}
	content := strings.TrimSpace(strings.Join(sections, "\n\n"))
	if content == "" {
		return service.ParsedDocument{}, fmt.Errorf("document is empty")
	}
	return service.ParsedDocument{Content: content, Title: firstNonEmptyLine(content)}, nil
}

type officeArchiveReader struct {
	archive  *zip.Reader
	xmlBytes uint64
	images   int
}

func newOfficeArchiveReader(archive *zip.Reader) *officeArchiveReader {
	return &officeArchiveReader{archive: archive}
}

func (r *officeArchiveReader) readXML(name string) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("document archive is missing")
	}
	remaining := maxOfficeXMLTotalBytes
	if r.xmlBytes >= maxOfficeXMLTotalBytes {
		return nil, fmt.Errorf("document archive expanded content is too large")
	}
	remaining -= r.xmlBytes
	limit := maxOfficeXMLEntryBytes
	if remaining < limit {
		limit = remaining
	}
	data, err := r.readFile(name, limit)
	if err != nil {
		return nil, err
	}
	r.xmlBytes += uint64(len(data))
	return data, nil
}

func (r *officeArchiveReader) readOCRImage(name string) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("document archive is missing")
	}
	if r.images >= maxPPTXOCRImages {
		return nil, fmt.Errorf("presentation contains too many images for OCR")
	}
	data, err := r.readFile(name, maxPPTXOCRImageBytes)
	if err != nil {
		return nil, err
	}
	r.images++
	return data, nil
}

func (r *officeArchiveReader) remainingOCRImages() int {
	if r == nil || r.images >= maxPPTXOCRImages {
		return 0
	}
	return maxPPTXOCRImages - r.images
}

func (r *officeArchiveReader) readFile(name string, limit uint64) ([]byte, error) {
	if r == nil || r.archive == nil {
		return nil, fmt.Errorf("document archive is missing")
	}
	for _, file := range r.archive.File {
		if file.Name != name {
			continue
		}
		if file.UncompressedSize64 > limit {
			return nil, fmt.Errorf("document archive entry is too large")
		}
		reader, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("document archive entry could not be opened")
		}
		defer reader.Close()
		data, err := io.ReadAll(io.LimitReader(reader, int64(limit)+1))
		if err != nil {
			return nil, fmt.Errorf("document archive entry could not be read")
		}
		if uint64(len(data)) > limit {
			return nil, fmt.Errorf("document archive entry is too large")
		}
		return data, nil
	}
	return nil, errArchiveEntryMissing
}

func sortedArchiveFiles(archive *officeArchiveReader, prefix string, suffix string) []string {
	files := []string{}
	if archive == nil || archive.archive == nil {
		return files
	}
	for _, file := range archive.archive.File {
		if strings.HasPrefix(file.Name, prefix) && strings.HasSuffix(file.Name, suffix) && !strings.HasSuffix(file.Name, "/") {
			files = append(files, file.Name)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		left := trailingNumber(files[i])
		right := trailingNumber(files[j])
		if left != right {
			return left < right
		}
		return files[i] < files[j]
	})
	return files
}

func orderedPresentationSlides(archive *officeArchiveReader) []string {
	fallback := sortedArchiveFiles(archive, "ppt/slides/", ".xml")
	presentation, err := archive.readXML("ppt/presentation.xml")
	if err != nil {
		return fallback
	}
	rels := relationshipsFor(archive, "ppt/_rels/presentation.xml.rels", "ppt")
	if len(rels) == 0 {
		return fallback
	}
	ids, err := orderedRelationshipIDs(presentation, "sldId")
	if err != nil {
		return fallback
	}
	files := filesFromRelationships(ids, rels)
	if len(files) == 0 {
		return fallback
	}
	return files
}

func orderedWorkbookSheets(archive *officeArchiveReader) []string {
	fallback := sortedArchiveFiles(archive, "xl/worksheets/", ".xml")
	workbook, err := archive.readXML("xl/workbook.xml")
	if err != nil {
		return fallback
	}
	rels := relationshipsFor(archive, "xl/_rels/workbook.xml.rels", "xl")
	if len(rels) == 0 {
		return fallback
	}
	ids, err := orderedRelationshipIDs(workbook, "sheet")
	if err != nil {
		return fallback
	}
	files := filesFromRelationships(ids, rels)
	if len(files) == 0 {
		return fallback
	}
	return files
}

func relationshipsFor(archive *officeArchiveReader, relsPath string, baseDir string) map[string]string {
	data, err := archive.readXML(relsPath)
	if err != nil {
		return nil
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	rels := map[string]string{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		id := attrValue(start.Attr, "Id")
		target := attrValue(start.Attr, "Target")
		if id == "" || target == "" || strings.Contains(target, "://") {
			continue
		}
		rels[id] = normalizeArchiveTarget(baseDir, target)
	}
	return rels
}

func imageTargetsForSlide(archive *officeArchiveReader, slideFile string) []string {
	slideData, err := archive.readXML(slideFile)
	if err != nil {
		return nil
	}
	referencedIDs := referencedImageRelationshipIDs(slideData)
	if len(referencedIDs) == 0 {
		return nil
	}
	relsPath := path.Dir(slideFile) + "/_rels/" + path.Base(slideFile) + ".rels"
	data, err := archive.readXML(relsPath)
	if err != nil {
		return nil
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	targets := []string{}
	seen := map[string]bool{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}
		id := attrValue(start.Attr, "Id")
		relationshipType := attrValue(start.Attr, "Type")
		target := attrValue(start.Attr, "Target")
		targetMode := attrValue(start.Attr, "TargetMode")
		if id == "" || !referencedIDs[id] || target == "" || strings.Contains(target, "://") || strings.EqualFold(targetMode, "External") {
			continue
		}
		normalized := normalizeArchiveTarget(path.Dir(slideFile), target)
		if !strings.HasSuffix(relationshipType, "/image") || !isImageArchivePath(normalized) || !strings.HasPrefix(normalized, "ppt/media/") {
			continue
		}
		if normalized != "" && !seen[normalized] {
			targets = append(targets, normalized)
			seen[normalized] = true
		}
	}
	return targets
}

func referencedImageRelationshipIDs(data []byte) map[string]bool {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	ids := map[string]bool{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		for _, attr := range start.Attr {
			if (attr.Name.Local == "embed" || attr.Name.Local == "link") &&
				strings.Contains(attr.Name.Space, "relationships") &&
				strings.TrimSpace(attr.Value) != "" {
				ids[attr.Value] = true
			}
		}
	}
	return ids
}

func isImageArchivePath(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff", ".webp":
		return true
	default:
		return false
	}
}

func orderedRelationshipIDs(data []byte, elementName string) ([]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	ids := []string{}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("document XML could not be parsed")
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != elementName {
			continue
		}
		id := relationshipID(start.Attr)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func filesFromRelationships(ids []string, rels map[string]string) []string {
	files := []string{}
	for _, id := range ids {
		file := rels[id]
		if file != "" {
			files = append(files, file)
		}
	}
	return files
}

func relationshipID(attrs []xml.Attr) string {
	for _, attr := range attrs {
		if attr.Name.Local == "id" && strings.Contains(attr.Name.Space, "relationships") {
			return attr.Value
		}
	}
	return ""
}

func normalizeArchiveTarget(baseDir string, target string) string {
	target = strings.TrimSpace(strings.TrimPrefix(target, "/"))
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, baseDir+"/") {
		return path.Clean(target)
	}
	return path.Clean(baseDir + "/" + target)
}

func trailingNumber(name string) int {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	end := len(base)
	start := end
	for start > 0 && base[start-1] >= '0' && base[start-1] <= '9' {
		start--
	}
	if start == end {
		return 0
	}
	value, err := strconv.Atoi(base[start:end])
	if err != nil {
		return 0
	}
	return value
}

func extractParagraphText(data []byte) ([]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	paragraphs := []string{}
	var current strings.Builder
	inParagraph := false
	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("document XML could not be parsed")
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "p":
				if !inParagraph {
					inParagraph = true
					current.Reset()
				}
			case "t":
				inText = true
			}
		case xml.CharData:
			if inText {
				current.WriteString(string(typed))
			}
		case xml.EndElement:
			switch typed.Name.Local {
			case "t":
				inText = false
			case "p":
				if inParagraph {
					text := strings.TrimSpace(current.String())
					if text != "" {
						paragraphs = append(paragraphs, text)
					}
					inParagraph = false
					current.Reset()
				}
			}
		}
	}
	return paragraphs, nil
}

func parseSharedStrings(archive *officeArchiveReader) ([]string, error) {
	if archive == nil || archive.archive == nil || !hasZipFile(archive.archive, "xl/sharedStrings.xml") {
		return nil, nil
	}
	data, err := archive.readXML("xl/sharedStrings.xml")
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	values := []string{}
	var current strings.Builder
	inString := false
	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("shared strings XML could not be parsed")
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "si":
				inString = true
				current.Reset()
			case "t":
				inText = true
			}
		case xml.CharData:
			if inString && inText {
				current.WriteString(string(typed))
			}
		case xml.EndElement:
			switch typed.Name.Local {
			case "t":
				inText = false
			case "si":
				values = append(values, strings.TrimSpace(current.String()))
				inString = false
				current.Reset()
			}
		}
	}
	return values, nil
}

func extractWorksheetRows(data []byte, sharedStrings []string) ([]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	rows := []string{}
	cells := []string{}
	inRow := false
	inCell := false
	inValue := false
	cellType := ""
	var value strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("worksheet XML could not be parsed")
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "row":
				inRow = true
				cells = nil
			case "c":
				inCell = true
				cellType = attrValue(typed.Attr, "t")
				value.Reset()
			case "v", "t":
				if inCell {
					inValue = true
				}
			}
		case xml.CharData:
			if inCell && inValue {
				value.WriteString(string(typed))
			}
		case xml.EndElement:
			switch typed.Name.Local {
			case "v", "t":
				inValue = false
			case "c":
				text := resolveCellValue(cellType, strings.TrimSpace(value.String()), sharedStrings)
				if text != "" {
					cells = append(cells, text)
				}
				inCell = false
				cellType = ""
				value.Reset()
			case "row":
				if inRow && len(cells) > 0 {
					rows = append(rows, strings.Join(cells, "\t"))
				}
				inRow = false
				cells = nil
			}
		}
	}
	return rows, nil
}

func resolveCellValue(cellType string, raw string, sharedStrings []string) string {
	if raw == "" {
		return ""
	}
	if cellType == "s" {
		index, err := strconv.Atoi(raw)
		if err != nil || index < 0 || index >= len(sharedStrings) {
			return ""
		}
		return sharedStrings[index]
	}
	return raw
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func firstNonEmptyLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
