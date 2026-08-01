/*
 * Copyright 2026 Holger de Carne
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package markdown

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/tdrn-org/mnemosyne/internal/crypto"
	"github.com/tdrn-org/mnemosyne/internal/domain"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type Parser struct {
	md        goldmark.Markdown
	tokenizer Tokenizer
	Debug     bool
}

func NewParser(tokenizer Tokenizer) *Parser {
	p := &Parser{
		md:        goldmark.New(),
		tokenizer: tokenizer,
	}
	return p
}

func (p *Parser) Parse(path string, source []byte, tokenLimit int) ([]domain.Chunk, error) {
	decoder := &chunkDecoder{
		parser:        p,
		parserContext: parser.NewContext(),
		path:          path,
		source:        source,
	}
	decoder.Decode()
	return decoder.Chunk(tokenLimit), nil
}

type chunkDecoder struct {
	parser         *Parser
	parserContext  parser.Context
	path           string
	source         []byte
	rootSection    *decodedSection
	currentSection *decodedSection
}

type decodedSection struct {
	Title       string
	HeadingPath headingPath
	Blocks      []decodedBlock
	Children    []*decodedSection
	TokenCount  int
}

type decodedBlock struct {
	Content    string
	TokenCount int
}

func (d *chunkDecoder) Decode() {
	d.rootSection = &decodedSection{
		HeadingPath: make(headingPath, 0),
		Blocks:      make([]decodedBlock, 0),
		Children:    make([]*decodedSection, 0),
	}
	d.currentSection = d.rootSection
	reader := text.NewReader(d.source)
	rootNode := d.parser.md.Parser().Parse(reader, parser.WithContext(d.parserContext))
	ast.Walk(rootNode, d.walk)
}

func (d *chunkDecoder) walk(node ast.Node, entering bool) (ast.WalkStatus, error) {
	if d.parser.Debug {
		if entering {
			fmt.Printf(">%T\n", node)
		} else {
			fmt.Printf("<%T\n", node)
		}
	}
	if !entering {
		return ast.WalkContinue, nil
	}
	switch node := node.(type) {
	case *ast.Heading:
		return d.walkHeading(node)
	case *ast.Paragraph:
		return d.walkParagraph(node)
	case *ast.TextBlock:
		return d.walkTextBlock(node)
	case *ast.List:
		return d.walkList(node)
	}
	return ast.WalkContinue, nil
}

func (d *chunkDecoder) walkHeading(heading *ast.Heading) (ast.WalkStatus, error) {
	d.newSection(heading)
	return ast.WalkSkipChildren, nil
}

func (d *chunkDecoder) walkBaseBlock(baseBlock *ast.BaseBlock) (ast.WalkStatus, error) {
	content := strings.TrimSpace(string(baseBlock.Lines().Value(d.source)))
	tokenCount := d.parser.tokenizer.Count(content)
	d.currentSection.Blocks = append(d.currentSection.Blocks, decodedBlock{
		Content:    content,
		TokenCount: tokenCount,
	})
	d.currentSection.TokenCount += tokenCount
	return ast.WalkSkipChildren, nil
}

func (d *chunkDecoder) walkParagraph(paragraph *ast.Paragraph) (ast.WalkStatus, error) {
	return d.walkBaseBlock(&paragraph.BaseBlock)
}

func (d *chunkDecoder) walkTextBlock(textBlock *ast.TextBlock) (ast.WalkStatus, error) {
	return d.walkBaseBlock(&textBlock.BaseBlock)
}

func (d *chunkDecoder) walkList(list *ast.List) (ast.WalkStatus, error) {
	return d.walkBaseBlock(&list.BaseBlock)
}

func (d *chunkDecoder) newSection(heading *ast.Heading) {
	title := strings.TrimSpace(string(heading.Lines().Value(d.source)))
	headingPath := append(headingPath{}, d.currentSection.HeadingPath...)
	headingPath = append(headingPath, d.currentSection.Title)
	headingPath.SetLevel(heading.Level - 1)
	d.currentSection = &decodedSection{
		Title:       title,
		HeadingPath: headingPath,
		Blocks:      make([]decodedBlock, 0),
		Children:    make([]*decodedSection, 0),
	}
	d.rootSection.addChild(d.currentSection, 0)
}

func (s *decodedSection) addChild(child *decodedSection, level int) {
	s.TokenCount += child.TokenCount
	if len(child.HeadingPath) == level {
		s.Children = append(s.Children, child)
		return
	}
	for _, next := range s.Children {
		if next.Title == child.HeadingPath[level] {
			next.addChild(child, level+1)
			return
		}
	}
	next := &decodedSection{
		HeadingPath: child.HeadingPath[:level],
		Blocks:      make([]decodedBlock, 0),
		Children:    make([]*decodedSection, 0),
	}
	s.Children = append(s.Children, next)
	next.addChild(child, level+1)
}

func (d *chunkDecoder) Chunk(tokenLimit int) []domain.Chunk {
	aggregator := &chunkAggregator{
		path:       d.path,
		tokenLimit: tokenLimit,
		chunks:     make([]domain.Chunk, 0),
	}
	chunks := aggregator.Chunk(d.rootSection)
	return chunks
}

type chunkAggregator struct {
	path       string
	tokenLimit int
	chunks     []domain.Chunk
}

func (a *chunkAggregator) Chunk(section *decodedSection) []domain.Chunk {
	a.chunkHelper(section)
	return a.chunks
}

func (a *chunkAggregator) chunkHelper(section *decodedSection) {
	if section.TokenCount < a.tokenLimit {
		a.addSectionFull(section)
	} else {
		a.addSectionShallow(section)
		for _, child := range section.Children {
			a.chunkHelper(child)
		}
	}
}

func (a *chunkAggregator) addSectionFull(section *decodedSection) {
	id := a.chunkID(section, 0)
	content := a.renderFull(section)
	chunkHash := crypto.HashString(content)
	a.chunks = append(a.chunks, domain.Chunk{
		ID:            id,
		Path:          a.path,
		ChunkIndex:    int64(len(a.chunks)),
		ChunkHash:     chunkHash,
		DocumentTitle: section.Title,
		HeadingPath:   section.HeadingPath,
		Tags:          make([]string, 0),
		Links:         make([]string, 0),
		Content:       content,
	})
}

func (a *chunkAggregator) renderFull(section *decodedSection) string {
	buffer := &strings.Builder{}
	for _, block := range section.Blocks {
		a.writeBlock(buffer, block)
	}
	for _, child := range section.Children {
		a.renderFullHelper(buffer, child)
	}
	return buffer.String()
}

func (a *chunkAggregator) renderFullHelper(buffer *strings.Builder, section *decodedSection) {
	buffer.WriteString(strings.Repeat("#", len(section.HeadingPath)+1))
	buffer.WriteString(section.Title)
	buffer.WriteRune('\n')
	for _, block := range section.Blocks {
		a.writeBlock(buffer, block)
	}
	for _, child := range section.Children {
		a.renderFullHelper(buffer, child)
	}
}

func (a *chunkAggregator) addSectionShallow(section *decodedSection) {
	id := a.chunkID(section, 0)
	content := a.renderShallow(section)
	chunkHash := crypto.HashString(content)
	a.chunks = append(a.chunks, domain.Chunk{
		ID:            id,
		Path:          a.path,
		ChunkIndex:    int64(len(a.chunks)),
		ChunkHash:     chunkHash,
		DocumentTitle: section.Title,
		HeadingPath:   section.HeadingPath,
		Tags:          make([]string, 0),
		Links:         make([]string, 0),
		Content:       content,
	})
}

func (a *chunkAggregator) renderShallow(section *decodedSection) string {
	buffer := &strings.Builder{}
	for _, block := range section.Blocks {
		a.writeBlock(buffer, block)
	}
	return buffer.String()
}

func (a *chunkAggregator) chunkID(section *decodedSection, subIndex int) string {
	idPath := bytes.Buffer{}
	idPath.WriteString(a.path)
	for _, heading := range section.HeadingPath {
		idPath.WriteString(heading)
	}
	idPath.WriteString(section.Title)
	idPath.WriteString(strconv.Itoa(subIndex))
	return uuid.NewSHA1(uuid.NameSpaceURL, idPath.Bytes()).String()
}

func (a *chunkAggregator) writeBlock(buffer *strings.Builder, block decodedBlock) {
	buffer.WriteString(strings.TrimSpace(block.Content))
	buffer.WriteString("\n\n")
}

type headingPath []string

func (p *headingPath) String() string {
	buffer := &strings.Builder{}
	for i, heading := range *p {
		if buffer.Len() > 0 {
			buffer.WriteString(" > ")
		}
		level := i + 1
		for range level {
			buffer.WriteRune('#')
		}
		buffer.WriteString(heading)
	}
	return buffer.String()
}

func (p *headingPath) SetLevel(level int) {
	if level < 0 {
		return
	}
	for len(*p) < level {
		*p = append(*p, "")
	}
	*p = (*p)[:level]
}
