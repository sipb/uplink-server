// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package markdown

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInspect(t *testing.T) {
	markdown := `
[foo]: bar
- a
  > [![]()]()
  > [![foo]][foo]
- d
`

	visited := []string{}
	level := 0
	Inspect(markdown, func(blockOrInline interface{}) bool {
		if blockOrInline == nil {
			level--
		} else {
			visited = append(visited, strings.Repeat(" ", level*4)+strings.TrimPrefix(fmt.Sprintf("%T", blockOrInline), "*markdown."))
			level++
		}
		return true
	})

	assert.Equal(t, []string{
		"Document",
		"    Paragraph",
		"    List",
		"        ListItem",
		"            Paragraph",
		"                Text",
		"            BlockQuote",
		"                Paragraph",
		"                    InlineLink",
		"                        InlineImage",
		"                    SoftLineBreak",
		"                    ReferenceLink",
		"                        ReferenceImage",
		"                            Text",
		"        ListItem",
		"            Paragraph",
		"                Text",
	}, visited)
}

var counterSink int

func BenchmarkInspect(b *testing.B) {
	text := `Some standard piece of text.

Has a link [post](https://github.com) and also has a blockquote.

> This is a famous quote.

Some bold text **Text for markdown?** to go with it.

At the end, some more lines`

	for i := 0; i < b.N; i++ {
		Inspect(text, func(_ interface{}) bool {
			counterSink++
			return true
		})
	}
}
