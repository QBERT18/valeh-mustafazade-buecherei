# 📚 Bücherei

Eine kleine Bücherei-API in **Go**, mit Mini-Frontend in **React**.

> [!TIP]
> **Schnellstart:** `.env` anlegen ([so geht's](#1-env-anlegen)), dann:
>
> ```sh
> docker compose up -d --build
> ```
>
> Danach im Browser **http://localhost:3000** öffnen. Beispieldaten sind schon da.

## Inhalt

- [Worum geht es?](#-worum-geht-es)
- [Starten](#-starten)
- [Docker-Befehle](#-docker-befehle)
- [API testen](#-api-testen) (Go-Tests und Postman)
- [Frontend](#-frontend)
- [Fragen und Antworten](#-fragen-und-antworten)
- [Wie ich weitermachen würde](#-wie-ich-weitermachen-würde)

---

## 💡 Worum geht es?

| | Regel |
|---|---|
| 🗄️ **Regale** | haben ein Limit, wie viele Bücher darauf passen. |
| 📖 **Bücher** | stehen auf einem Regal (oder auf keinem) und können eine PDF- oder EPUB-Datei haben. |
| 🙋 **Personen** | können Bücher ausleihen und zurückgeben. |
| 🔒 **Präsenzbestand** | (z. B. Lexika) darf gelesen, aber nicht ausgeliehen werden. |

Die Daten (Regale, Bücher, Personen, Ausleihen) liegen in **Postgres**.
Die Dateien der Bücher liegen in **RustFS**, einem S3-Speicher wie MinIO. In der Datenbank steht nur, wo die Datei liegt.

| Endpunkt | Was er macht |
|---|---|
| `/shelves` | Regale anlegen, lesen, ändern, löschen. Mit [Suche und Sortierung](#suchen-und-sortieren). |
| `/books` | Bücher anlegen, lesen, ändern, löschen. Mit [Suche und Sortierung](#suchen-und-sortieren). |
| `/books/:id/file` | Datei hochladen (`PUT`), herunterladen (`GET`), löschen (`DELETE`) |
| `/books/:id/loan` · `/books/:id/return` | Buch ausleihen · zurückgeben |
| `/people` | Personen anlegen, lesen, ändern, löschen |
| `/loans` | Alle Ausleihen ansehen (Verlauf) |

---

## 🚀 Starten

> [!NOTE]
> **Voraussetzung:** Docker. Go 1.27 und Node brauchst du nur, wenn du App, Frontend oder Tests ohne Docker laufen lässt.

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

Das startet die App, das Frontend, Postgres und RustFS, legt den Bucket `books` an und lädt die Beispieldaten.

| Was | Adresse |
|---|---|
| 🖥️ Frontend | http://localhost:3000 |
| ⚙️ API | http://localhost:8080 |
| 🗂️ RustFS-Oberfläche | http://localhost:9001/rustfs/console/ (Login: `rustfsadmin` / `rustfsadmin`) |
| 🐘 Postgres | `localhost:5432` |

### 3. Beispieldaten

Die Beispieldaten werden beim Start **automatisch** geladen (Service `seed`):

| | |
|---|---|
| 🗄️ 5 Regale | „Moderne“ ist schon voll. |
| 📖 25 Bücher | jedes mit einer kleinen PDF. 3 sind Präsenzbestand, 2 stehen auf keinem Regal. |
| 🙋 4 Personen | Anna, Ben, Clara und David |
| 🔁 7 Ausleihen | 4 sind noch offen, 3 schon zurückgegeben. |

> [!NOTE]
> Wenn schon Daten da sind, passiert beim Seeding nichts. Frische Beispieldaten bekommst du nur über einen [Reset](#zurücksetzen-und-beispieldaten).

### 4. Ausprobieren

```sh
curl localhost:8080/books
curl "localhost:8080/books?q=kafka&sort=-jahr"
curl -X PUT localhost:8080/books/1/file -F "file=@tests/testdata/test.pdf"
```

### App ohne Docker starten

```sh
docker compose up -d postgres rustfs storage-init
go run .
```

Die App liest dann die Werte aus `.env`.

---

## 🐳 Docker-Befehle

### Starten und stoppen

| Befehl | Was er macht |
|---|---|
| `docker compose up -d --build` | Alles bauen und starten. **Nach Änderungen am Code genau das noch mal ausführen.** |
| `docker compose up -d` | Starten, ohne neu zu bauen |
| `docker compose up -d --build app` | Nur die App neu bauen und starten |
| `docker compose up -d --build frontend` | Nur das Frontend neu bauen und starten |
| `docker compose restart app` | App neu starten, ohne neu zu bauen |
| `docker compose stop` | Alles anhalten. Die Daten bleiben. |
| `docker compose down` | Alles anhalten und die Container löschen. Die Daten bleiben. |

### Zurücksetzen und Beispieldaten

| Befehl | Was er macht |
|---|---|
| `docker compose down -v && docker compose up -d --build` | 🔄 **Reset:** löscht alle Daten (Datenbank und Dateien) und startet neu mit frischen Beispieldaten |
| `docker compose run --rm seed` | Beispieldaten von Hand laden. Das wirkt nur, wenn die Datenbank leer ist. |

> [!CAUTION]
> `-v` löscht die Volumes `postgres-data` und `rustfs-data`. **Alles, was du selbst angelegt hast, ist danach weg.**

### Nachsehen

| Befehl | Was er macht |
|---|---|
| `docker compose ps` | Welche Container laufen? `seed` und `storage-init` stehen auf „Exited“, das ist richtig. |
| `docker compose logs -f app` | Logs der App live ansehen (Strg+C zum Beenden) |
| `docker compose logs seed` | Was hat das Seeding gemacht? |
| `docker compose exec postgres psql -U buecherei` | In die Datenbank schauen, z. B. `SELECT * FROM books;` (mit `\q` beenden) |

---

## 🧪 API testen

Es gibt zwei Wege: die **Go-Tests** für Entwickler und eine **Postman-Collection**, mit der man die laufende API von außen durchklickt.

### Go-Tests

```sh
docker compose up -d   # die Tests brauchen Postgres und RustFS
go test ./...
```

> [!NOTE]
> Die Tests nutzen eine eigene Datenbank (`buecherei_test`) und einen eigenen Bucket (`books-test`). Deine normalen Daten bleiben unberührt.

### Postman-Collection

Die Collection liegt in [`postman/buecherei.postman_collection.json`](postman/buecherei.postman_collection.json). Sie hat **54 Requests** und prüft bei jedem den Statuscode, oft auch die Antwort.

> [!IMPORTANT]
> Die Collection ist eine **Reihenfolge**. Sie legt ihre eigenen Testdaten an, merkt sich deren IDs in Variablen und räumt am Ende wieder auf. Lass sie deshalb als Ganzes **von oben nach unten** laufen.

**In Postman (mit Oberfläche):**

1. Stack starten: `docker compose up -d --build`
2. In Postman **Import** → die Datei `postman/buecherei.postman_collection.json` wählen.
3. **Settings → General → Working directory** auf den Projektordner setzen. Sonst findet der Upload-Request die Testdatei `tests/testdata/test.pdf` nicht.
4. Collection anklicken → **Run** → **Run Bewerbertag**.

**Im Terminal (mit [Newman](https://www.npmjs.com/package/newman)):**

```sh
npx newman run postman/buecherei.postman_collection.json
```

Im Projektordner ausführen, damit die Testdatei gefunden wird. Eine andere Adresse geht mit `--env-var baseUrl=http://...`.

**Was wird geprüft?**

| Ordner | Inhalt |
|---|---|
| 1 Shelves | Regal anlegen, lesen, suchen, sortieren, ändern |
| 2 Books | Buch anlegen (auch Präsenzbestand ohne Regal), lesen, suchen, sortieren, ändern |
| 3 Book files | PDF hochladen, herunterladen, löschen |
| 4 People | Person anlegen, lesen, ändern |
| 5 Loans | Ausleihen und zurückgeben, mit den Regeln: nicht doppelt ausleihen, kein Präsenzbestand, ausgeliehene Bücher und Personen mit Büchern nicht löschen |
| 6 Shelf limit | Volles Regal (`409`), Limit nicht unter die Anzahl der Bücher (`409`) |
| 7 Invalid input | Falsche Eingaben: `400`, `404`, `415` |
| 8 Cleanup | Löscht alles, was die Collection angelegt hat |

> [!TIP]
> Die Collection läuft gegen deine normalen Daten, ändert sie aber nicht. Übrig bleibt nur ein Eintrag im Ausleih-Verlauf (`/loans`), weil Ausleihen nie gelöscht werden.

---

## 💻 Frontend

Ein kleines Frontend mit **React**, **TypeScript** und **Tailwind**. Es liegt im Ordner `frontend/`.

### Starten

Mit Docker läuft es automatisch mit (`docker compose up -d --build`) auf **http://localhost:3000**.

Ohne Docker (die API muss auf `localhost:8080` laufen):

```sh
cd frontend
npm install
npm run dev    # http://localhost:5173
npm test       # Tests für slugify
```

### Seiten

| Adresse | Was man dort macht |
|---|---|
| `/books` | Alle Bücher. Suchen und sortieren. |
| `/books/new` | Neues Buch anlegen, optional mit PDF oder EPUB |
| `/books/1-faust` | Ein Buch ansehen, Datei herunterladen, ausleihen, zurückgeben, löschen |
| `/books/1-faust/edit` | Ein Buch bearbeiten und die Datei ersetzen |
| `/shelves`, `/shelves/new`, `/shelves/1-klassiker`, `/shelves/1-klassiker/edit` | Das Gleiche für Regale |
| `/me` | Meine Ausleihen: was ich gerade habe, und was ich schon zurückgegeben habe |

### Wer bin ich?

> [!NOTE]
> Es gibt **kein echtes Login**. Beim ersten Besuch fragt die Seite nach deinem Namen und legt dich als Person an. Der Browser merkt sich deine ID (`localStorage`). Mit „Abmelden“ vergisst er sie wieder, und du kannst einen neuen Namen eingeben.

### Suchen und Sortieren

Das macht das **Backend**, nicht der Browser (warum, steht unten bei „Pagination und Sorting“).

```sh
curl "localhost:8080/books?q=goethe&sort=-jahr"
curl "localhost:8080/shelves?sort=name"
```

| Parameter | Bedeutung |
|---|---|
| `q` | Sucht im Titel, Autor und in der ISBN (bei Regalen: im Namen). Groß- und Kleinschreibung ist egal. |
| `sort` | Bücher: `titel`, `autor`, `jahr`, `created`. Regale: `name`, `limit`. Ein `-` davor heißt absteigend, z. B. `-jahr`. Andere Werte geben `400`. |

Suche und Sortierung stehen auch in der Adresse der Seite (`/books?q=goethe&sort=-jahr`). So kann man eine Liste als Lesezeichen speichern.

### Slugs

Die Adressen enthalten den Namen, damit man sie lesen kann: aus dem Buch 1 „Faust. Eine Tragödie“ wird `/books/1-faust-eine-tragoedie`. Umlaute werden zu `ae`, `oe`, `ue`, und `ß` wird zu `ss`.

Die Seite liest nur die **Zahl vorne**. Ändert sich der Titel, funktioniert der alte Link also weiter. Die Seite tauscht die Adresse dann gegen die neue aus.

### Wie spricht das Frontend mit der API?

Das Frontend ruft immer `/api/...` auf, zum Beispiel `/api/books`.

- **Im Container** leitet nginx das an die App weiter (`app:8080/books`).
- **Mit `npm run dev`** macht das Vite.

Frontend und API sehen für den Browser so aus, als kämen sie vom selben Server. Deshalb braucht das Backend **kein CORS**.

---

## ❓ Fragen und Antworten

Klick auf eine Frage, um die Antwort zu sehen.

<details>
<summary><b>Warum Postgres und nicht MongoDB?</b></summary>

Weil unsere Daten **zusammenhängen** und **Regeln** haben:

- Ein Regal hat viele Bücher, eine Person hat viele Ausleihen, eine Ausleihe gehört zu genau einem Buch und einer Person.
- „Ein Regal darf nicht zu voll werden.“
- „Ein Buch darf nur an eine Person gleichzeitig ausgeliehen sein.“

Genau dafür ist eine relationale Datenbank wie Postgres gemacht. Sie verbindet Tabellen und prüft solche Regeln selbst. Dass ein Buch nicht doppelt ausgeliehen wird, garantiert bei uns die Datenbank. Selbst wenn im Code ein Fehler wäre, ginge es nicht.

Postgres kann außerdem mehrere Änderungen **gemeinsam oder gar nicht** speichern (eine *Transaktion*). Ein Beispiel: Wird ein Regal gelöscht, nehmen wir in einem Schritt alle Bücher herunter und löschen dann das Regal. Entweder klappt beides oder nichts.

MongoDB ist gut für Daten ohne feste Form, die kaum miteinander verbunden sind. Das ist hier nicht der Fall. Mit MongoDB müssten wir viele dieser Regeln selbst im Code bauen.

</details>

<details>
<summary><b>Was passiert beim Hochladen einer Datei?</b></summary>

1. Die App prüft die Datei:
   - Ist sie größer als **50 MB**, antwortet sie mit `413`.
   - Ist sie keine **PDF oder EPUB**, antwortet sie mit `415`. Die App schaut dafür in die Datei selbst und verlässt sich nicht auf die Dateiendung.
   - Gibt es das **Buch** nicht, antwortet sie mit `404`.
2. Die Datei wird in RustFS gespeichert, und zwar immer unter einem **neuen, eindeutigen Namen** (`books/<buch-id>/<zufall>.pdf`). Es wird nie eine bestehende Datei überschrieben.
3. **Erst danach** schreibt die App in die Datenbank, wo die neue Datei liegt.
4. Hatte das Buch vorher schon eine Datei, wird die alte jetzt gelöscht.

> [!IMPORTANT]
> Die wichtigste Regel dahinter: **Die Datenbank zeigt nie auf eine Datei, die es nicht gibt.** Andersherum ist es harmlos, wenn irgendwo eine Datei liegt, auf die nichts zeigt.

</details>

<details>
<summary><b>Und wenn die Verbindung weg ist?</b></summary>

| Was bricht ab? | Was passiert? |
|---|---|
| Nutzer → App (z. B. Internet weg) | Die App speichert erst, wenn die ganze Datei angekommen ist. Bricht der Upload vorher ab, wird nichts gespeichert. Der Nutzer lädt einfach noch mal hoch. |
| App → RustFS | Das Speichern schlägt fehl und die App antwortet mit einem Fehler. In der Datenbank ändert sich nichts. |
| App → Datenbank | Die Datei liegt schon in RustFS, aber die Datenbank kennt sie nicht. Die App löscht die Datei deshalb wieder. |
| App stürzt genau zwischen Schritt 2 und 3 ab | Dann bleibt eine Datei übrig, auf die nichts zeigt. Das ist harmlos. Ein kleiner Aufräum-Job könnte solche Dateien ab und zu löschen (noch nicht gebaut). |

In keinem Fall zeigt ein Buch auf eine kaputte oder fehlende Datei.

</details>

<details>
<summary><b>Zu große Dateien?</b></summary>

Im Moment gilt ein Limit von **50 MB**. Alles darüber lehnt die App sofort mit `413` ab.

Ist das irgendwann zu wenig, gibt es den nächsten Schritt: **Direkt-Upload mit einem temporären Link**.

- Die App gibt dem Nutzer einen Link, der ein paar Minuten gültig ist.
- Der Nutzer lädt die Datei damit direkt in RustFS hoch, an der App vorbei.
- Große Dateien werden dabei in Stücken hochgeladen. Bricht ein Stück ab, wird nur dieses Stück wiederholt.

So muss die App die großen Dateien nicht selbst durchreichen.

</details>

<details>
<summary><b>Brauchen wir eine Queue?</b></summary>

**Für den Upload selbst: nein.** Der Upload ist schnell, und der Nutzer will sofort wissen, ob er geklappt hat.

**Für Arbeit, die lange dauert: ja, später.** Zum Beispiel:

- Dateien auf Viren prüfen
- Vorschaubilder erstellen
- den Text aus der PDF lesen, damit man darin suchen kann

Diese Arbeit kommt dann in eine Queue (eine Warteschlange). Ein Hintergrund-Prozess arbeitet sie Stück für Stück ab, und der Nutzer muss nicht warten. Fällt dabei etwas aus, bleibt die Aufgabe in der Queue und wird später erneut versucht.

</details>

<details>
<summary><b>Zu viele Reads? Redis?</b></summary>

Erst einmal **nicht nötig**. Vorher gibt es einfachere Schritte:

1. **Indizes**: Damit findet Postgres Daten schnell, ohne jede Zeile anzuschauen. Wir haben schon welche, z. B. auf `shelf_id` und `book_id`.
2. **Postgres merkt sich selbst** oft gelesene Daten im Arbeitsspeicher.
3. **Verbindungs-Pool**: Die App öffnet nicht für jede Anfrage eine neue Verbindung, sondern nutzt bis zu 25 offene Verbindungen immer wieder.
4. **Lese-Kopien** der Datenbank: Weitere Postgres-Server, die nur Lese-Anfragen beantworten.

Erst wenn das nicht reicht, kommt **Redis** als Zwischenspeicher davor, z. B. für die Bücherliste.

Für die **Dateien** brauchen wir Redis nicht. Die kommen direkt aus RustFS oder später aus einem CDN.

</details>

<details>
<summary><b>Was passiert bei einem Cache-Miss?</b></summary>

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

</details>

<details>
<summary><b>Zu viele Writes?</b></summary>

In einer Bücherei passiert das praktisch nicht. Geschrieben wird nur, wenn jemand ein Buch anlegt, ausleiht oder zurückgibt. Das sind wenige Aktionen pro Minute. Postgres schafft Tausende pro Sekunde.

</details>

<details>
<summary><b>Zu viele Bücher? Sharding oder größere Datenbank?</b></summary>

**Erst die größere Datenbank. Sharding ganz zum Schluss.**

Zur Größenordnung: Ein Buch ist in der Datenbank nur eine kleine Zeile, weniger als 1 KB, weil die Datei selbst in RustFS liegt. **10 Millionen Bücher** sind damit nur wenige GB. Das schafft ein einzelner Postgres-Server problemlos.

Die Reihenfolge, wenn es wirklich eng wird:

1. **Indizes** prüfen und **langsame Abfragen** verbessern.
2. **Größerer Server**: mehr Arbeitsspeicher, schnellere Festplatte. Einfach und meistens genug.
3. **Partitionierung**: Eine große Tabelle wird intern in Teile geteilt, z. B. Ausleihen nach Jahr. Für die App sieht es weiter wie eine Tabelle aus.
4. **Sharding**: Die Daten auf mehrere Datenbank-Server verteilen. Das ist sehr aufwendig. Regeln wie „Regal nicht zu voll“ sind dann schwer, wenn Regal und Bücher auf verschiedenen Servern liegen. Deshalb nur, wenn wirklich nichts anderes mehr hilft.

Die **Dateien** wachsen viel schneller als die Datenbank. Dafür gibt man RustFS einfach mehr Festplatten oder weitere Server.

</details>

<details>
<summary><b>Pagination und Sorting: Frontend oder Backend? Und wie?</b></summary>

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
- **Sortieren** über einen Parameter, z. B. `?sort=titel` oder `?sort=-jahr` (das Minus heißt absteigend). *Sortieren und Suchen sind schon gebaut, siehe [Suchen und Sortieren](#suchen-und-sortieren). Pagination noch nicht.*
- Nur **erlaubte Spalten** zum Sortieren zulassen (Titel, Autor, Jahr). Den Wert aus der URL nie direkt in die SQL-Abfrage schreiben, sonst ist das eine Sicherheitslücke (SQL-Injection).
- Für jede Spalte, nach der sortiert wird, einen **Index** anlegen, damit es schnell bleibt.

</details>

---

## 🧭 Wie ich weitermachen würde

Die App ist ein guter Anfang. Bevor ich weiterbaue, würde ich mit der Bücherei ein paar Fragen klären. Die Antworten entscheiden, was als Nächstes wichtig ist und was wir gar nicht brauchen.

> [!TIP]
> Jede Frage hat zwei Teile: was die App **heute** macht, und was sich je nach **Antwort** ändert.

### Erst fragen

#### 📊 Wie viele Bücher wird es geben?

**Heute:** Alle Listen kommen komplett zurück. Die Suche nutzt `ILIKE '%…%'`. Dafür kann Postgres keinen normalen Index nutzen und schaut jede Zeile an.

| Antwort | Was ich dann mache |
|---|---|
| ein paar Tausend | Nichts. Das passt so. |
| Zehntausende und mehr | **Pagination** (Cursor, siehe Frage „Pagination und Sorting“) und für die Suche einen **Trigramm-Index** (`pg_trgm`), damit sie schnell bleibt |
| Millionen | Eine eigene **Suchmaschine** (z. B. OpenSearch) für die Suche. Die Datenbank selbst schafft das noch (siehe „Zu viele Bücher?“). |

Genauso wichtig: **Wie viele Leute nutzen die App gleichzeitig?** Davon hängt ab, ob wir mehrere App-Server und einen Cache brauchen (siehe „Zu viele Reads?“).

#### 📏 Wie groß sind die Dateien?

**Heute:** höchstens **50 MB**. Der Upload läuft durch die App.

| Antwort | Was ich dann mache |
|---|---|
| normale E-Books (EPUB 1–5 MB, Text-PDFs) | Nichts. Das passt so. |
| eingescannte Bücher, Bildbände (oft 200 MB und mehr) | **Direkt-Upload** mit temporärem Link, in Stücken (siehe „Zu große Dateien?“) |

Und die Gesamtmenge: 100.000 Bücher × 20 MB sind schon **2 TB**. Das muss man beim Speicher und bei den Backups einplanen.

#### 📄 Welche Formate wird es geben?

**Heute:** **PDF und EPUB**, genau eine Datei pro Buch. Die App prüft den Inhalt der Datei, nicht die Endung.

| Antwort | Was ich dann mache |
|---|---|
| auch Hörbücher (MP3, M4B) | Dateien werden viel größer. Abspielen im Browser braucht Streaming (der Player lädt nur das Stück, das gerade läuft). |
| mehrere Formate pro Buch (PDF **und** EPUB) | Eine eigene Tabelle für Dateien statt der Felder am Buch |
| im Browser lesen statt herunterladen | Ein Reader im Frontend (z. B. PDF.js, epub.js) |
| gekaufte E-Books mit Lizenz | Dann darf nicht jeder herunterladen. Ein E-Book wird dann wie ein echtes Buch **ausgeliehen** (eine Lizenz = ein Leser gleichzeitig). |

#### 👥 Verschiedene Nutzer? Rechte?

**Heute:** Es gibt **kein Login**. Jeder darf alles, auch löschen. Das Frontend merkt sich die Person nur im Browser.

> [!WARNING]
> Das ist die größte Lücke der App. Für echte Nutzer muss das als Erstes kommen.

Mein Vorschlag für Rollen:

| Rolle | Darf |
|---|---|
| 👀 Gast | Bücher ansehen und suchen |
| 📖 Leser | dazu: ausleihen, zurückgeben, eigene Ausleihen sehen, Dateien herunterladen |
| 🗂️ Bibliothekar | dazu: Bücher und Regale anlegen, ändern, löschen; Ausleihen für andere buchen |
| 🔑 Admin | dazu: Nutzer und Rollen verwalten |

Wichtige Frage dazu: **Gibt es schon Konten**, z. B. von der Schule, der Stadt oder der Firma? Dann melden sich die Nutzer damit an (Single Sign-on über OpenID Connect), und wir speichern keine eigenen Passwörter.

Technisch prüft dann eine Middleware bei jeder Anfrage, wer angemeldet ist und welche Rolle er hat. Beim Ausleihen kommt die Person aus dem Login und nicht mehr aus dem Request.

#### 📚 Ein Titel, mehrere Exemplare?

**Heute:** ein Buch = ein Exemplar. Hat die Bücherei „Faust“ dreimal, sind das drei Bücher mit drei Einträgen.

Wenn es mehrere Exemplare gibt, würde ich das trennen:

- **Titel:** Titel, Autor, ISBN, Datei. Gibt es einmal.
- **Exemplar:** steht auf einem Regal, wird ausgeliehen. Gibt es so oft wie im Bestand.

> [!IMPORTANT]
> Das ist ein großer Umbau, der fast alle Endpunkte betrifft. Deshalb würde ich diese Frage **zuerst** klären.

#### ⏰ Wie lange darf man ausleihen?

**Heute:** Es gibt kein Rückgabedatum. Man darf so viele Bücher ausleihen, wie man will.

Fragen an die Bücherei:

- Gibt es eine **Frist** (z. B. 4 Wochen)? Darf man **verlängern**?
- Soll es eine **Erinnerung** per E-Mail geben? **Mahngebühren**?
- Wie viele Bücher darf eine Person **gleichzeitig** haben?
- Kann man ein ausgeliehenes Buch **vormerken** (Warteliste)?

Dafür bräuchte die Ausleihe ein Feld „zurück bis“, und ein Job prüft jeden Tag, was überfällig ist. Hier passt die Queue aus „Brauchen wir eine Queue?“.

#### 🔐 Datenschutz

**Heute:** Der Ausleih-Verlauf wird **für immer** gespeichert. Gelöschte Personen sind nur als gelöscht markiert und bleiben in der Datenbank.

Wer was gelesen hat, sind persönliche Daten (DSGVO). Fragen dazu:

- Wie lange darf der Verlauf gespeichert werden?
- Muss eine Person auf Wunsch **wirklich** gelöscht werden können?

Mein Vorschlag: Den Verlauf nach einer festen Zeit **anonymisieren**. Die Person wird entfernt, die Zahlen für eine Statistik („wie oft wurde Faust ausgeliehen?“) bleiben.

#### 💬 Weitere Fragen

| Frage | Warum |
|---|---|
| Gibt es **mehrere Standorte**? | Dann gehört jedes Regal zu einem Standort, und man sucht pro Standort. |
| Sollen Buchdaten **per ISBN** automatisch geladen werden? | Spart Tipparbeit, z. B. über die Deutsche Nationalbibliothek. |
| Gibt es **alte Daten** zum Übernehmen? | Dann brauchen wir einen Import, z. B. aus Excel oder dem alten System. |
| **Wo** soll die App laufen? | Eigener Server oder Cloud entscheidet über Backups, Updates und Kosten. |
| Wie schnell muss sie nach einem **Ausfall** wieder laufen? | Davon hängt ab, wie oft wir Backups machen und ob wir einen zweiten Server brauchen. |
| Muss die Seite **barrierefrei** sein? | Für öffentliche Stellen ist das Pflicht (BITV). |
| Nur **Deutsch**? | Mehrere Sprachen sind später schwer nachzurüsten. |

### Dann bauen

Unabhängig von den Antworten würde ich in dieser Reihenfolge weitermachen:

| # | Was | Warum |
|---|---|---|
| 1 | 🔑 **Login und Rollen** | Heute kann jeder alles löschen. |
| 2 | 🤖 **Automatische Tests bei jedem Push** (CI) | Go-Tests, Frontend-Tests und die Postman-Collection laufen dann von selbst. |
| 3 | 📄 **Pagination** | Bevor die Listen groß werden. Dazu ein Endpunkt für die offene Ausleihe eines Buchs: Die Buch-Seite lädt dafür heute alle Ausleihen. |
| 4 | 💾 **Backups** für Postgres und RustFS | Und einmal ausprobieren, ob man sie auch wieder einspielen kann. |
| 5 | 📈 **Überwachung** | Ein `/health`-Endpunkt und Kennzahlen, damit man Probleme sieht, bevor die Nutzer sie melden |
| 6 | ⏰ **Ausleihfristen** | Sobald die Regeln geklärt sind |
| 7 | 🧹 **Aufräum-Job** für Dateien ohne Buch | Siehe „Und wenn die Verbindung weg ist?“ |
