package seed

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/QBERT18/valeh-mustafazade-buecherei/storage"
	"github.com/minio/minio-go/v7"
	"golang.org/x/text/encoding/charmap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func year(y int) datatypes.Date {
	return datatypes.Date(time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC))
}

// Run fills an empty database with sample shelves and books, each book with a
// one-page PDF in storage. It does nothing if any shelf or book already exists.
func Run(ctx context.Context, g *gorm.DB, st *storage.Storage) error {
	var shelves, books int64
	if err := g.Model(&db.Shelf{}).Count(&shelves).Error; err != nil {
		return err
	}
	if err := g.Model(&db.Book{}).Count(&books).Error; err != nil {
		return err
	}
	if shelves > 0 || books > 0 {
		log.Printf("database already has data (%d shelves, %d books), skipping seed", shelves, books)
		return nil
	}

	klassiker := db.Shelf{ShelfName: "Klassiker", Limit: 5}
	moderne := db.Shelf{ShelfName: "Moderne", Limit: 3}
	seedBooks := []db.Book{
		{Titel: "Faust. Eine Tragödie", Autor: "Johann Wolfgang von Goethe", Jahr: year(1808)},
		{Titel: "Effi Briest", Autor: "Theodor Fontane", Jahr: year(1896)},
		{Titel: "Die Verwandlung", Autor: "Franz Kafka", Jahr: year(1915)},
		{Titel: "Der Steppenwolf", Autor: "Hermann Hesse", Jahr: year(1927)}, // stays without a shelf
	}

	err := g.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&klassiker).Error; err != nil {
			return err
		}
		if err := tx.Create(&moderne).Error; err != nil {
			return err
		}
		seedBooks[0].ShelfID = &klassiker.ID
		seedBooks[1].ShelfID = &klassiker.ID
		seedBooks[2].ShelfID = &moderne.ID
		return tx.Create(&seedBooks).Error
	})
	if err != nil {
		return err
	}

	// Same order as the upload endpoint: store the file, then point the row at it.
	for i := range seedBooks {
		b := &seedBooks[i]
		pdf, err := onePagePDF(fmt.Sprintf("%s - %s", b.Titel, b.Autor))
		if err != nil {
			return err
		}
		key := storage.BookFileKey(b.ID, ".pdf")
		if _, err := st.Client.PutObject(ctx, st.Bucket, key, bytes.NewReader(pdf), int64(len(pdf)),
			minio.PutObjectOptions{ContentType: "application/pdf"}); err != nil {
			return err
		}
		b.FileKey = &key
		b.FileName = fileName(b.Titel) + ".pdf"
		b.FileContentType = "application/pdf"
		b.FileSize = int64(len(pdf))
		if err := g.Save(b).Error; err != nil {
			return err
		}
	}

	log.Printf("seeded 2 shelves and %d books with PDFs", len(seedBooks))
	return nil
}

// fileName turns a title into a simple file name, e.g. "Effi Briest" -> "effi-briest".
func fileName(title string) string {
	r := strings.NewReplacer(".", "", " ", "-", "ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")
	return r.Replace(strings.ToLower(title))
}

// onePagePDF builds a minimal valid PDF with one line of text.
func onePagePDF(text string) ([]byte, error) {
	// Helvetica with WinAnsiEncoding expects Windows-1252 bytes (so umlauts work).
	encoded, err := charmap.Windows1252.NewEncoder().String(text)
	if err != nil {
		return nil, err
	}
	escaped := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(encoded)
	content := fmt.Sprintf("BT /F1 18 Tf 72 720 Td (%s) Tj ET", escaped)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
	}

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes(), nil
}
