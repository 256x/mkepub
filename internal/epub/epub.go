package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"mkepub/internal/aozora"
)

type Book struct {
	Title      string
	Author     string
	Language   string
	CoverPNG   []byte
	Paragraphs []string
	Chapters   []aozora.Chapter
}

func Build(b *Book) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// mimetype must be first and uncompressed
	mw, err := w.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	})
	if err != nil {
		return nil, err
	}
	mw.Write([]byte("application/epub+zip"))

	// META-INF/container.xml
	if err := addFile(w, "META-INF/container.xml", containerXML()); err != nil {
		return nil, err
	}

	// Cover image
	if b.CoverPNG != nil {
		if err := addBytes(w, "OEBPS/images/cover.png", b.CoverPNG); err != nil {
			return nil, err
		}
	}

	// cover.xhtml
	if err := addFile(w, "OEBPS/cover.xhtml", coverXHTML(b.Title, b.Author)); err != nil {
		return nil, err
	}

	// content.xhtml
	if err := addFile(w, "OEBPS/content.xhtml", contentXHTML(b.Title, b.Author, b.Paragraphs)); err != nil {
		return nil, err
	}

	// stylesheet
	if err := addFile(w, "OEBPS/style.css", css()); err != nil {
		return nil, err
	}

	// content.opf
	if err := addFile(w, "OEBPS/content.opf", opf(b)); err != nil {
		return nil, err
	}

	// toc.ncx
	if err := addFile(w, "OEBPS/toc.ncx", ncx(b)); err != nil {
		return nil, err
	}

	w.Close()
	return buf.Bytes(), nil
}

func xmlEscape(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

func addFile(w *zip.Writer, name, content string) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

func addBytes(w *zip.Writer, name string, data []byte) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func containerXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
}

func opf(b *Book) string {
	lang := b.Language
	if lang == "" {
		lang = "ja"
	}
	uid := fmt.Sprintf("mkepub-%d", time.Now().UnixNano())
	hasCover := b.CoverPNG != nil

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="BookID">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>` + xmlEscape(b.Title) + `</dc:title>
    <dc:creator>` + xmlEscape(b.Author) + `</dc:creator>
    <dc:language>` + lang + `</dc:language>
    <dc:identifier id="BookID">` + uid + `</dc:identifier>`)
	sb.WriteString(`
    <meta name="primary-writing-mode" content="vertical-rl"/>`)
	if hasCover {
		sb.WriteString(`
    <meta name="cover" content="cover-image"/>`)
	}
	sb.WriteString(`
  </metadata>
  <manifest>`)
	if hasCover {
		sb.WriteString(`
    <item id="cover-image" href="images/cover.png" media-type="image/png"/>
    <item id="cover" href="cover.xhtml" media-type="application/xhtml+xml"/>`)
	}
	sb.WriteString(`
    <item id="content" href="content.xhtml" media-type="application/xhtml+xml"/>
    <item id="css" href="style.css" media-type="text/css"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx" page-progression-direction="rtl">`)
	if hasCover {
		sb.WriteString(`
    <itemref idref="cover" linear="no"/>`)
	}
	sb.WriteString(`
    <itemref idref="content"/>
  </spine>
</package>`)
	return sb.String()
}

func ncx(b *Book) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE ncx PUBLIC "-//NISO//DTD ncx 2005-1//EN" "http://www.daisy.org/z3986/2005/ncx-2005-1.dtd">
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="mkepub"/>
    <meta name="dtb:depth" content="1"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle><text>` + xmlEscape(b.Title) + `</text></docTitle>
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>` + xmlEscape(b.Title) + `</text></navLabel>
      <content src="content.xhtml"/>
    </navPoint>`)

	for i, ch := range b.Chapters {
		sb.WriteString(fmt.Sprintf(`
    <navPoint id="np%d" playOrder="%d">
      <navLabel><text>%s</text></navLabel>
      <content src="content.xhtml#%s"/>
    </navPoint>`, i+2, i+2, ch.Title, ch.Anchor))
	}

	sb.WriteString(`
  </navMap>
</ncx>`)
	return sb.String()
}

func coverXHTML(title, author string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>Cover</title>
  <style type="text/css">body{margin:0;padding:0;} img{max-width:100%;}</style>
</head>
<body>
  <div style="text-align:center;">
    <img src="images/cover.png" alt="` + title + ` - ` + author + `"/>
  </div>
</body>
</html>`
}

func contentXHTML(title, author string, paragraphs []string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="ja">
<head>
  <meta http-equiv="Content-Type" content="application/xhtml+xml; charset=UTF-8"/>
  <title>` + xmlEscape(title) + `</title>
  <link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
  <h1>` + xmlEscape(title) + `</h1>
  <p class="author">` + xmlEscape(author) + `</p>
`)
	for _, p := range paragraphs {
		sb.WriteString("  ")
		sb.WriteString(p)
		sb.WriteString("\n")
	}
	sb.WriteString("</body>\n</html>")
	return sb.String()
}

func css() string {
	return `body {
  writing-mode: vertical-rl;
  -webkit-writing-mode: vertical-rl;
  font-family: serif;
  line-height: 1.8;
  margin: 1em 1.5em;
}
h1 {
  font-size: 1.4em;
  margin-bottom: 0.3em;
}
h2 {
  font-size: 1.1em;
  margin-top: 2em;
}
p {
  text-indent: 1em;
  margin: 0.3em 0;
}
p.author {
  text-indent: 0;
  font-size: 0.9em;
  margin-bottom: 2em;
}
ruby rt {
  font-size: 0.5em;
}
hr.pagebreak {
  border: none;
  page-break-after: always;
}
`
}
