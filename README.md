# Bücherei

Eine kleine Bücherei-API in Go.

- **Regale** haben ein Limit, wie viele Bücher darauf passen.
- **Bücher** stehen auf einem Regal (oder auf keinem) und können eine PDF- oder EPUB-Datei haben.
- **Personen** können Bücher ausleihen und zurückgeben.
- **Präsenzbestand** (z. B. Lexika) darf gelesen, aber nicht ausgeliehen werden.

Die Daten (Regale, Bücher, Personen, Ausleihen) liegen in **Postgres**.
Die Dateien der Bücher liegen in **RustFS**, einem S3-Speicher wie MinIO. In der Datenbank steht nur, wo die Datei liegt.

| Endpunkt | Was er macht |
|---|---|
| `/shelves` | Regale anlegen, lesen, ändern, löschen |
| `/books` | Bücher anlegen, lesen, ändern, löschen |
| `/books/:id/file` | Datei hochladen (`PUT`), herunterladen (`GET`), löschen (`DELETE`) |
| `/books/:id/loan` · `/books/:id/return` | Buch ausleihen · zurückgeben |
| `/people` | Personen anlegen, lesen, ändern, löschen |
| `/loans` | Alle Ausleihen ansehen (Verlauf) |

---

## Starten und testen

**Voraussetzung:** Docker. Go 1.27 nur, wenn du die App oder die Tests ohne Docker laufen lässt.

### 1. `.env` anlegen

Lege im Projektordner eine Datei `.env` an. Git ignoriert sie, deshalb ist sie nicht im Repo.

```env
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=buecherei
POSTGRES_PASSWORD=buecherei
POSTGRES_DB=buecherei

S3_ENDPOINT=localhost:9000
S3_USE_SSL=false
S3_ACCESS_KEY=rustfsadmin
S3_SECRET_KEY=rustfsadmin
S3_BUCKET=books
```

### 2. Alles starten

```sh
docker compose up -d --build
```

Das startet die App, Postgres und RustFS und legt den Bucket `books` an.

| Was | Adresse |
|---|---|
| API | http://localhost:8080 |
| RustFS-Oberfläche | http://localhost:9001/rustfs/console/ (Login: `rustfsadmin` / `rustfsadmin`) |
| Postgres | `localhost:5432` |

### 3. Beispieldaten laden (optional)

```sh
docker compose run --rm app buecherei seed
```

Das legt 2 Regale und 4 Bücher an, jedes mit einer kleinen PDF. Wenn schon Daten da sind, passiert nichts.

### 4. Ausprobieren

```sh
curl localhost:8080/books
curl -X PUT localhost:8080/books/1/file -F "file=@tests/testdata/test.pdf"
```

### Tests

```sh
docker compose up -d   # die Tests brauchen Postgres und RustFS
go test ./...
```

Die Tests nutzen eine eigene Datenbank (`buecherei_test`) und einen eigenen Bucket (`books-test`). Deine normalen Daten bleiben unberührt.

### App ohne Docker starten

```sh
docker compose up -d postgres rustfs storage-init
go run .
```

Die App liest dann die Werte aus `.env`.

---

## Fragen und Antworten

### Warum Postgres und nicht MongoDB?

Weil unsere Daten **zusammenhängen** und **Regeln** haben:

- Ein Regal hat viele Bücher, eine Person hat viele Ausleihen, eine Ausleihe gehört zu genau einem Buch und einer Person.
- „Ein Regal darf nicht zu voll werden.“
- „Ein Buch darf nur an eine Person gleichzeitig ausgeliehen sein.“

Genau dafür ist eine relationale Datenbank wie Postgres gemacht. Sie verbindet Tabellen und prüft solche Regeln selbst. Dass ein Buch nicht doppelt ausgeliehen wird, garantiert bei uns die Datenbank. Selbst wenn im Code ein Fehler wäre, ginge es nicht.

Postgres kann außerdem mehrere Änderungen **gemeinsam oder gar nicht** speichern (eine *Transaktion*). Ein Beispiel: Wird ein Regal gelöscht, nehmen wir in einem Schritt alle Bücher herunter und löschen dann das Regal. Entweder klappt beides oder nichts.

MongoDB ist gut für Daten ohne feste Form, die kaum miteinander verbunden sind. Das ist hier nicht der Fall. Mit MongoDB müssten wir viele dieser Regeln selbst im Code bauen.

---

### Was passiert beim Hochladen einer Datei?

1. Die App prüft die Datei:
   - Ist sie größer als **50 MB**, antwortet sie mit `413`.
   - Ist sie keine **PDF oder EPUB**, antwortet sie mit `415`. Die App schaut dafür in die Datei selbst und verlässt sich nicht auf die Dateiendung.
   - Gibt es das **Buch** nicht, antwortet sie mit `404`.
2. Die Datei wird in RustFS gespeichert, und zwar immer unter einem **neuen, eindeutigen Namen** (`books/<buch-id>/<zufall>.pdf`). Es wird nie eine bestehende Datei überschrieben.
3. **Erst danach** schreibt die App in die Datenbank, wo die neue Datei liegt.
4. Hatte das Buch vorher schon eine Datei, wird die alte jetzt gelöscht.

Die wichtigste Regel dahinter: **Die Datenbank zeigt nie auf eine Datei, die es nicht gibt.** Andersherum ist es harmlos, wenn irgendwo eine Datei liegt, auf die nichts zeigt.

---

### Und wenn die Verbindung weg ist?

| Was bricht ab? | Was passiert? |
|---|---|
| Nutzer → App (z. B. Internet weg) | Die App speichert erst, wenn die ganze Datei angekommen ist. Bricht der Upload vorher ab, wird nichts gespeichert. Der Nutzer lädt einfach noch mal hoch. |
| App → RustFS | Das Speichern schlägt fehl und die App antwortet mit einem Fehler. In der Datenbank ändert sich nichts. |
| App → Datenbank | Die Datei liegt schon in RustFS, aber die Datenbank kennt sie nicht. Die App löscht die Datei deshalb wieder. |
| App stürzt genau zwischen Schritt 2 und 3 ab | Dann bleibt eine Datei übrig, auf die nichts zeigt. Das ist harmlos. Ein kleiner Aufräum-Job könnte solche Dateien ab und zu löschen (noch nicht gebaut). |

In keinem Fall zeigt ein Buch auf eine kaputte oder fehlende Datei.

---

### Zu große Dateien?

Im Moment gilt ein Limit von **50 MB**. Alles darüber lehnt die App sofort mit `413` ab.

Ist das irgendwann zu wenig, gibt es den nächsten Schritt: **Direkt-Upload mit einem temporären Link**.

- Die App gibt dem Nutzer einen Link, der ein paar Minuten gültig ist.
- Der Nutzer lädt die Datei damit direkt in RustFS hoch, an der App vorbei.
- Große Dateien werden dabei in Stücken hochgeladen. Bricht ein Stück ab, wird nur dieses Stück wiederholt.

So muss die App die großen Dateien nicht selbst durchreichen.

---

### Brauchen wir eine Queue?

**Für den Upload selbst: nein.** Der Upload ist schnell, und der Nutzer will sofort wissen, ob er geklappt hat.

**Für Arbeit, die lange dauert: ja, später.** Zum Beispiel:

- Dateien auf Viren prüfen
- Vorschaubilder erstellen
- den Text aus der PDF lesen, damit man darin suchen kann

Diese Arbeit kommt dann in eine Queue (eine Warteschlange). Ein Hintergrund-Prozess arbeitet sie Stück für Stück ab, und der Nutzer muss nicht warten. Fällt dabei etwas aus, bleibt die Aufgabe in der Queue und wird später erneut versucht.

---

### Zu viele Reads? Redis?

Erst einmal **nicht nötig**. Vorher gibt es einfachere Schritte:

1. **Indizes**: Damit findet Postgres Daten schnell, ohne jede Zeile anzuschauen. Wir haben schon welche, z. B. auf `shelf_id` und `book_id`.
2. **Postgres merkt sich selbst** oft gelesene Daten im Arbeitsspeicher.
3. **Verbindungs-Pool**: Die App öffnet nicht für jede Anfrage eine neue Verbindung, sondern nutzt bis zu 25 offene Verbindungen immer wieder.
4. **Lese-Kopien** der Datenbank: Weitere Postgres-Server, die nur Lese-Anfragen beantworten.

Erst wenn das nicht reicht, kommt **Redis** als Zwischenspeicher davor, z. B. für die Bücherliste.

Für die **Dateien** brauchen wir Redis nicht. Die kommen direkt aus RustFS oder später aus einem CDN.

---

### Was passiert bei einem Cache-Miss?

Ein *Cache-Miss* heißt: Die Daten sind (noch) nicht in Redis.

1. Die App fragt Redis → nichts da.
2. Die App holt die Daten aus **Postgres**.
3. Sie legt die Daten in Redis ab, mit einem Ablaufdatum (z. B. 5 Minuten).
4. Sie gibt die Daten zurück. Die nächste Anfrage bekommt sie direkt aus Redis.

**Ein Miss ist also kein Fehler.** Die Anfrage ist nur etwas langsamer, weil sie zur Datenbank geht.

Zwei Dinge muss man beachten:

- **Veraltete Daten:** Wenn sich ein Buch ändert, muss der passende Eintrag in Redis gelöscht werden. Sonst sieht man noch eine Weile die alte Version.
- **Viele Misses gleichzeitig:** Läuft ein beliebter Eintrag ab und 1000 Anfragen kommen im selben Moment, würden alle 1000 die Datenbank fragen. Lösung: Nur eine Anfrage holt die Daten, die anderen warten kurz auf ihr Ergebnis.

Und wenn Redis ganz ausfällt, fragt die App einfach direkt Postgres. Es wird langsamer, aber es geht nichts kaputt.

---

### Zu viele Writes?

In einer Bücherei passiert das praktisch nicht. Geschrieben wird nur, wenn jemand ein Buch anlegt, ausleiht oder zurückgibt. Das sind wenige Aktionen pro Minute. Postgres schafft Tausende pro Sekunde.

---

### Zu viele Bücher? Sharding oder größere Datenbank?

**Erst die größere Datenbank. Sharding ganz zum Schluss.**

Zur Größenordnung: Ein Buch ist in der Datenbank nur eine kleine Zeile, weniger als 1 KB, weil die Datei selbst in RustFS liegt. **10 Millionen Bücher** sind damit nur wenige GB. Das schafft ein einzelner Postgres-Server problemlos.

Die Reihenfolge, wenn es wirklich eng wird:

1. **Indizes** prüfen und **langsame Abfragen** verbessern.
2. **Größerer Server**: mehr Arbeitsspeicher, schnellere Festplatte. Einfach und meistens genug.
3. **Partitionierung**: Eine große Tabelle wird intern in Teile geteilt, z. B. Ausleihen nach Jahr. Für die App sieht es weiter wie eine Tabelle aus.
4. **Sharding**: Die Daten auf mehrere Datenbank-Server verteilen. Das ist sehr aufwendig. Regeln wie „Regal nicht zu voll“ sind dann schwer, wenn Regal und Bücher auf verschiedenen Servern liegen. Deshalb nur, wenn wirklich nichts anderes mehr hilft.

Die **Dateien** wachsen viel schneller als die Datenbank. Dafür gibt man RustFS einfach mehr Festplatten oder weitere Server.

---

### Pagination und Sorting: Frontend oder Backend? Und wie?

**Im Backend.**

Würde das Frontend sortieren und in Seiten aufteilen, müsste es vorher **alle** Bücher laden. Bei 4 Büchern ist das egal, bei 100.000 ist es langsam und verschwendet Daten. Das Backend schickt besser nur die Seite, die gerade gebraucht wird.

Es gibt zwei Arten von Pagination:

| | Offset („Seite 5“) | Cursor („weiter ab Buch 120“) |
|---|---|---|
| Aufruf | `GET /books?page=5&size=20` | `GET /books?after=120&size=20` |
| Vorteil | Einfach. Man kann direkt zu „Seite 5“ springen. | Bleibt schnell, egal wie weit man blättert. Keine doppelten oder fehlenden Einträge. |
| Nachteil | Weit hinten wird es langsam, weil die Datenbank alle Einträge davor überspringen muss. Kommt ein neues Buch dazu, verrutschen die Seiten. | Man kann nicht direkt zu „Seite 5“ springen, nur vor und zurück. |
| Gut für | kleine Listen, z. B. Regale | lange Listen, z. B. Bücher oder Ausleihen |

**Unsere Empfehlung:**

- **Cursor-Pagination** für Bücher und Ausleihen. Die Antwort enthält einen Wert für „nächste Seite“, den das Frontend beim nächsten Aufruf mitschickt.
- **Sortieren** über einen Parameter, z. B. `?sort=titel` oder `?sort=-jahr` (das Minus heißt absteigend).
- Nur **erlaubte Spalten** zum Sortieren zulassen (Titel, Autor, Jahr). Den Wert aus der URL nie direkt in die SQL-Abfrage schreiben, sonst ist das eine Sicherheitslücke (SQL-Injection).
- Für jede Spalte, nach der sortiert wird, einen **Index** anlegen, damit es schnell bleibt.
