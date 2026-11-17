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

type seedShelf struct {
	name  string
	limit int
	books []db.Book
}

// Run fills an empty database with sample shelves, books (each with a one-page
// PDF in storage), people and loans. It does nothing if any shelf or book
// already exists.
func Run(ctx context.Context, g *gorm.DB, st *storage.Storage) error {
	var shelfCount, bookCount int64
	if err := g.Model(&db.Shelf{}).Count(&shelfCount).Error; err != nil {
		return err
	}
	if err := g.Model(&db.Book{}).Count(&bookCount).Error; err != nil {
		return err
	}
	if shelfCount > 0 || bookCount > 0 {
		log.Printf("database already has data (%d shelves, %d books), skipping seed", shelfCount, bookCount)
		return nil
	}

	shelves := []seedShelf{
		{"Klassiker", 8, []db.Book{
			{Titel: "Faust. Eine Tragödie", Autor: "Johann Wolfgang von Goethe", Jahr: year(1808)},
			{Titel: "Die Leiden des jungen Werthers", Autor: "Johann Wolfgang von Goethe", Jahr: year(1774)},
			{Titel: "Nathan der Weise", Autor: "Gotthold Ephraim Lessing", Jahr: year(1779)},
			{Titel: "Die Räuber", Autor: "Friedrich Schiller", Jahr: year(1781)},
			{Titel: "Der Schimmelreiter", Autor: "Theodor Storm", Jahr: year(1888)},
			{Titel: "Effi Briest", Autor: "Theodor Fontane", Jahr: year(1896)},
			{Titel: "Buddenbrooks", Autor: "Thomas Mann", Jahr: year(1901)},
		}},
		{"Moderne", 6, []db.Book{ // full: no more space
			{Titel: "Die Verwandlung", Autor: "Franz Kafka", Jahr: year(1915)},
			{Titel: "Der Process", Autor: "Franz Kafka", Jahr: year(1925)},
			{Titel: "Im Westen nichts Neues", Autor: "Erich Maria Remarque", Jahr: year(1929)},
			{Titel: "Die Blechtrommel", Autor: "Günter Grass", Jahr: year(1959)},
			{Titel: "Der Vorleser", Autor: "Bernhard Schlink", Jahr: year(1995)},
			{Titel: "Tschick", Autor: "Wolfgang Herrndorf", Jahr: year(2010)},
		}},
		{"Krimis", 5, []db.Book{
			{Titel: "Der Richter und sein Henker", Autor: "Friedrich Dürrenmatt", Jahr: year(1952)},
			{Titel: "Das Versprechen", Autor: "Friedrich Dürrenmatt", Jahr: year(1958)},
			{Titel: "Tannöd", Autor: "Andrea Maria Schenkel", Jahr: year(2006)},
		}},
		{"Kinderbücher", 5, []db.Book{
			{Titel: "Emil und die Detektive", Autor: "Erich Kästner", Jahr: year(1929)},
			{Titel: "Das doppelte Lottchen", Autor: "Erich Kästner", Jahr: year(1949)},
			{Titel: "Momo", Autor: "Michael Ende", Jahr: year(1973)},
			{Titel: "Die unendliche Geschichte", Autor: "Michael Ende", Jahr: year(1979)},
		}},
		{"Nachschlagewerke", 4, []db.Book{
			{Titel: "Duden. Die deutsche Rechtschreibung", Autor: "Konrad Duden", Jahr: year(1880), Praesenzbestand: true},
			{Titel: "Wahrig Deutsches Wörterbuch", Autor: "Gerhard Wahrig", Jahr: year(1966), Praesenzbestand: true},
			{Titel: "Brockhaus Enzyklopädie", Autor: "F. A. Brockhaus", Jahr: year(2006), Praesenzbestand: true},
		}},
	}
	withoutShelf := []db.Book{
		{Titel: "Siddhartha", Autor: "Hermann Hesse", Jahr: year(1922)},
		{Titel: "Der Steppenwolf", Autor: "Hermann Hesse", Jahr: year(1927)},
	}
	people := []db.Person{{Name: "Anna Schmidt"}, {Name: "Ben Müller"}, {Name: "Clara Weber"}, {Name: "David Fischer"}}

	daysAgo := func(n int) *time.Time {
		t := time.Now().AddDate(0, 0, -n)
		return &t
	}
	// person index, book title, lent n days ago, returned n days ago (nil = still on loan)
	loans := []struct {
		person   int
		title    string
		lent     int
		returned *time.Time
	}{
		{0, "Faust. Eine Tragödie", 60, daysAgo(30)},
		{0, "Effi Briest", 10, nil},
		{1, "Momo", 5, nil},
		{1, "Tschick", 3, nil},
		{2, "Die Verwandlung", 40, daysAgo(20)},
		{2, "Der Process", 18, daysAgo(2)},
		{2, "Das Versprechen", 1, nil},
	}

	var books []db.Book
	err := g.Transaction(func(tx *gorm.DB) error {
		for _, s := range shelves {
			shelf := db.Shelf{ShelfName: s.name, Limit: s.limit}
			if err := tx.Create(&shelf).Error; err != nil {
				return err
			}
			for _, b := range s.books {
				b.ShelfID = &shelf.ID
				books = append(books, b)
			}
		}
		books = append(books, withoutShelf...)
		if err := tx.Create(&books).Error; err != nil {
			return err
		}
		if err := tx.Create(&people).Error; err != nil {
			return err
		}

		idByTitle := map[string]uint{}
		for _, b := range books {
			idByTitle[b.Titel] = b.ID
		}
		for _, l := range loans {
			bookID, ok := idByTitle[l.title]
			if !ok {
				return fmt.Errorf("seed loan: unknown book %q", l.title)
			}
			loan := db.Loan{BookID: bookID, PersonID: people[l.person].ID, LoanedAt: *daysAgo(l.lent), ReturnedAt: l.returned}
			if err := tx.Create(&loan).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Same order as the upload endpoint: store the file, then point the row at it.
	for i := range books {
		b := &books[i]
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

	log.Printf("seeded %d shelves, %d books with PDFs, %d people and %d loans",
		len(shelves), len(books), len(people), len(loans))
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
