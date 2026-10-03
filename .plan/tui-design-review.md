# Emqutiti TUI/Design Review

## Ablage

Dies ist das urspruengliche Review vor der Implementierung. Status, offene
Designfragen und Testgrenzen unten beschreiben diesen Zeitpunkt. Die Umsetzung
ist in `lab-226-implementation.md` dokumentiert; die integrierte Abschlusspruefung
mit LAB-232/LAB-233 und Release ist in `lab-232-233-release.md` festgehalten.

- Projekt: [Emqutiti](https://linear.app/riotbox/project/emqutiti-6db064dcbd41), Team LAB.
- Issue: [LAB-226](https://linear.app/riotbox/issue/LAB-226/tuidesign-gemeinsam-abgestimmtes-review-von-codex-und-kimi), Backlog, Prioritaet High.
- Lokale Fassung des gemeinsam abgestimmten Review-Berichts.
- Kimi-Sitzung: `session_29aa5e60-2863-4d0c-8700-b412077ee80d`,
  Repo-CWD `/home/markus/Dev/emqutiti`, Modell `kimi-code/k3`, Thinking `high`.
- Der temporaere Review-Agent erlaubte nur Read/Grep/Glob/ReadMediaFile;
  keine Repository- oder externen Schreibaktionen durch Kimi.

# Gemeinsames TUI/Design-Review: Codex + Kimi

## Ergebnis und Scope

Review vom 2026-10-03 auf Codebasis `2d5a115e2b08353325d4a6f4e375698ae35633b8`.
Codex und die lokal installierte Kimi-CLI 2.0.2 haben das Review im Dialog
durchgefuehrt; tatsaechliches Modell `kimi-code/k3`, Thinking `high`,
im Sitzungsprofil verifiziert. Noch keine TUI-Fixes implementiert.

Abgedeckt: MQTT-Client und Message-Editor; Chips und Topics-Manager;
History, Filter und Details; Broker-Liste und Formular; Payloads;
Trace-Liste, Formular und Fehlerpfad; Logs, Hilfe und Bestaetigungsdialoge.
Code, vorhandene Tests und aktuelle Render-/Interaktions-Probes wurden
gemeinsam bewertet. Eine neue Animation oder Neugestaltung ist kein Ziel.

## P1: Zuerst Beheben

### 1. Publish-Ergebnis ehrlich und asynchron darstellen

**Belegt:** [client_keys.go:135](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/client_keys.go#L135) schreibt `pub`-History vor
dem MQTT-Aufruf, ignoriert dessen Fehler und schreibt auch ohne Client
einen vermeintlichen Publish. Die Handler starten Erfolgspulses ohne
Ergebnis. [mqttclient.go:37](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/mqttclient.go#L37) wartet synchron auf Tokens;
[update_client.go:44](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/update_client.go#L44) fuehrt auch Subscribe/Unsubscribe im
Update-Pfad aus. Langsame Verbindungen blockieren die TUI.

**Repro:** Ohne MQTT-Client, Ziel `probe/outgoing`, Payload
`offline payload`, Message fokussieren und Ctrl+S:
History enthaelt `kind=pub`, obwohl nichts gesendet wurde.

**Loesung:** MQTT-Arbeit in `tea.Cmd`, typisierte Ergebnisse je Ziel.
Pending, API-Erfolg und Fehler unterscheiden; Erfolgseintrag und Pulses
erst bei erfolgreichem Token-Ergebnis. Draft und Ziel-Snapshot erhalten,
Request/Broker-Identitaet gegen veraltete Ergebnisse absichern;
Worker mutieren den UI-Zustand nicht. QoS-0-Erfolg bedeutet keine
bestaetigte Zustellung an Subscriber. Subscription-Zustand nach Fehlern
ebenfalls mit dem Ergebnis abgleichen.

**Akzeptanz:** Offline, Timeout, Token-Fehler und Teilfehler erzeugen
keinen falschen Erfolg. Bei einem 5s-Timeout bleiben Navigation und
Eingabe bedienbar. Explizite Mehrfachziele, Fallback, retained Publish
und Brokerwechsel waehrend einer Anfrage sind getestet.

### 2. Payload-Mausaktionen duerfen nicht beliebige Eintraege treffen

**Belegt:** [payloads/payloads_component.go:82](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/payloads/payloads_component.go#L82) verwendet
die aktuelle Auswahl statt des angeklickten Eintrags.
Rechtsklick loescht sofort; Linksklick laedt und verlaesst die Ansicht.

**Repro:** Zwei Payloads, 80x24, Rechtsklick in die leere Ecke
`X=79,Y=23`: Der erste, ausgewaehlte Eintrag wird geloescht.

**Loesung:** Hit-Test gegen tatsaechlich gerenderte Listenzeilen;
Leerflaechen bleiben ohne Aktion. Loeschen konsistent ueber
Bestaetigung, mit stabiler Eintragsidentitaet statt einem fluechtigen Index.

**Akzeptanz:** Leerflaechen-Klick aendert nichts. Klick auf Zeile 2
laedt oder bestaetigt ausschliesslich Zeile 2; Esc bricht Loeschen ab.
Gefilterte/paginierte Listen und Delete waehrend Texteingabe mitpruefen.

### 3. Brokerformular innerhalb des Terminals bedienbar machen

**Belegt:** [view_form.go:11](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/view_form.go#L11) erzwingt zwei halbe Spalten;
[connections/form.go:200](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/connections/form.go#L200) rendert alle Felder ohne
Form-Viewport. Feldbreiten und Env-Hinweise sprengen das Budget.

**Repro:** Ausgabe 106x41 bei 80x24, 126x41 bei 120x40,
96x41 bei 60x20 und 86x41 bei 40x16. Captures zeigen abgeschnittene
Felder und umgebrochene Hinweise in der benachbarten Brokerliste.

**Loesung:** Adaptive Spalten, auf schmalen Terminals einspaltiges,
scrollbares Formular; Label- und Eingabebreiten aus dem verfuegbaren
Platz ableiten. Bestehende `ui.Form`-Helfer weiterverwenden.
Basic/Advanced-Gruppierung ist keine Voraussetzung.

**Akzeptanz:** Alle Felder samt Validierungsfehlern, Select-Optionen
und Speichern sind per Tastatur und Maus erreichbar.
Renderbreite/-hoehe ueberschreiten keine der vier Pruefgroessen;
Fokus scrollt das aktive Feld sichtbar, ohne Werte zu verlieren.

### 4. Suche und Topics-Kommandos strikt trennen

**Belegt:** [topics/component.go:62](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/topics/component.go#L62) verarbeitet
Aktionskeys vor `list.Update`, ohne den Filter-Eingabemodus abzugrenzen.

**Repro:** Nichtleere Topics-Liste, `/`, dann `p`:
Filter ist im Zustand `filtering`, aber `Items[0].Publish` wird
`true`. Sucheingabe aendert damit ausgehende Publish-Ziele.

**Loesung:** Aktive Texteingabe zuerst routen; erst ausserhalb dieses
Modus Topic-Aktionen erlauben. Entsprechendes Muster in Payloads und
Traces gezielt auditieren, nicht pauschal neue Shortcuts einfuehren.

**Akzeptanz:** Filtertext mit `p`, Leerzeichen und anderen Action-Letters
aendert keine Topic-Zustaende. Enter bestaetigt den Filter, Delete
editiert ihn. Nach Filterabschluss funktionieren Listenaktionen wieder
und treffen den sichtbaren Eintrag.

## P2: Bedienbarkeit und Fehlerpfade

### 5. Mauskoordinaten aus dem jeweiligen Layout ableiten

**Belegt:** [topics/component.go:98](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/topics/component.go#L98) verwendet im Manager
die alten Client-`ChipBounds`. [history/historyfilter.go:103](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/history/historyfilter.go#L103)
bildet `Y-1` auf Felder ab, obwohl Leer- und Suggestion-Zeilen
dazwischen liegen. [history/history_component.go:151](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/history/history_component.go#L151)
zentriert und polstert den Filter, ohne Klicks entsprechend umzurechnen.

**Repro:** Manager mit zwei subscribed Topics, Klick auf zweite
Zeile `X=5,Y=8`: Auswahl bleibt 0 statt 1.
Zentrierter Filter, Klick auf Start `X=33,Y=11`: Fokus bleibt 0
statt 2. Lokaler Start-Klick `Y=4` fokussiert End (3) statt Start (2).
Eine falsche Topic-Loeschung per Manager-Rechtsklick wurde nicht
reproduziert und ist nur ein codegestuetztes Risiko.

**Loesung/Akzeptanz:** Render-Geometrie fuer Hitboxes wiederverwenden;
Screen-, Pane- und Form-Offsets genau einmal uebersetzen.
Beide Topic-Panes, Pagination, Scroll, Leerflaechen, Blank- und
Suggestion-Zeilen testen. Klick und Tastatur treffen denselben Eintrag.

### 6. Fokus muss aktiven Inhalt zeigen, nicht nur den Boxtitel

**Belegt:** [focus.go:38](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/focus.go#L38) beruecksichtigt nur `pos-1`,
nicht Elementhoehe oder aktive Listenzeile.

**Repro:** History-Fokus bei 80x24 zeigt Titel und angeschnittene
erste Zeile ohne Payload; bei 60x20/40x16 nur die Titelkante.

**Loesung/Akzeptanz:** Bounds und aktive Zeile fuer EnsureVisible nutzen.
Mindestens aktiver Eintrag plus Payload sichtbar; bei zu grossen
Panels nicht blind ans Panel-Ende scrollen. Tab, Shift+Tab und Mausklick,
Rueckkehr aus Details, Draft und bestehende Scrollposition testen.
Header und zweizeilige Kontext-Hilfe bleiben oben stabil.

### 7. Verbleibende Layout-Ueberlaeufe konsistent behandeln

**Belegt:** [confirm/component.go:87](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/confirm/component.go#L87) begrenzt nur die
Boxbreite, nicht den Prompt. [topics/component.go:118](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/topics/component.go#L118),
[traces/view.go:10](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/traces/view.go#L10), [traces/view_form.go:8](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/traces/view_form.go#L8) und
[history/history_component.go:125](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/history/history_component.go#L125) brauchen ein
einheitliches Breiten-/Hoehenbudget.

**Repro:** Langer Topic-Loeschprompt rendert 102 Spalten bei 80/60/40.
Topics-Manager: 69 Spalten bei 60, 59 bei 40.
Leere Trace-Liste: 66 Spalten bei 60 und 40; Trace-Formular:
47 Spalten bei 40. History-Detail ist in allen Pruefgroessen eine
Zeile hoeher als das Terminal.

**Loesung/Akzeptanz:** Prompt sinnvoll umbrechen, lange Inhalte
scrollen; Listen/Forms adaptiv setzen und Header/Border/Hilfe mitrechnen.
Fuer jede Ansicht gilt Breite <= W und Hoehe <= H bei 120x40, 80x24,
60x20 und 40x16. Keine blosse Abschneidung von wichtigen Aktionen,
Dialogentscheidung oder Payload-Daten.

### 8. Fehler innerhalb der TUI melden, nicht auf stdout

**Von Kimi eingebracht, von Codex im Code bestaetigt:**
[history/history_component.go:204](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/history/history_component.go#L204) und
[traces/runtime.go:140](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/traces/runtime.go#L140) schreiben Fehler mit `fmt.Printf`.
Das kann den Fullscreen-Render stoeren; beim Trace-Subscribe-Fehler
fehlt zudem die strukturierte Fehlerweitergabe.

**Loesung/Akzeptanz:** History-Fehler ueber vorhandene Log-/Statuswege,
Trace-Fehler ueber `reportErr` und dann Logs/History.
Fehler absichtlich ausloesen: sichtbar, nachvollziehbar und ohne
unkoordinierten stdout-Output, der den Tea-Render zerstoert.

### 9. Kontext-Hilfe muss zum aktiven Eingabebereich passen

**Belegt:** [context_help.go:178](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/context_help.go#L178) gibt Topic-Input und
Topics-Chips dieselben Aktionshinweise.

**Repro:** Topic-Input `probe`, Taste `p`: Text wird korrekt
`probep`, Hinweis behauptet aber Publish-Umschalten.

**Loesung/Akzeptanz:** Input und Chips trennen. In den bestehenden zwei
Zeilen Zustand/Ergebnis und anwendbare Aktionen ohne doppelte Shortcuts
verteilen. Empty/Existing Topic, Hover, Tab-Fokus und alle Chip-Zustaende
testen. Publish-Ziele nutzen denselben expliziten/Fallback-Resolver wie
der Versand; kein irrefuehrendes `none`.

## P3: Gezielte Design-Experimente

- **Legende und Aktionen budgetieren:** [view_topics.go:119](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/view_topics.go#L119)
  kuerzt zuerst die Aktionen am Ende. Enter/Publish/Delete fehlen schon
  bei 80 Spalten; `Working` verkleinert das Budget weiter.
  Bestehende Legenden- und Hilfezeilen kompakter nutzen, keine neue Box.
  Relevante Aktionen bleiben bei 80 discoverable; definierte
  kompakte Darstellung bei 60/40 und waehrend Animationen.
- **Chip-Zustaende unabhaengig kodieren:** [view_topics.go:29](https://github.com/marang/emqutiti/blob/2d5a115e2b08353325d4a6f4e375698ae35633b8/view_topics.go#L29)
  macht Publish-only und Subscribe+Publish byte-identisch.
  Read-only, Write-only, beide und inactive muessen unterscheidbar sein,
  auch wenn Fokus oder Puls aktiv ist. Unterstrichene Namen und eine
  bestehende Border-Segment-Markierung als Varianten vergleichen;
  keine konkrete neue Gestaltung beschlossen. ANSI/No-color und
  stabile Abmessungen/Hitboxes mitpruefen.
- **Kontrast und Randzustaende:** Bestehende Palette auf hellem/dunklem
  Terminal pruefen; State nicht ausschliesslich farblich vermitteln.
  Leere, nicht verbundene, laufende und fehlerhafte Zustaende muessen
  korrekt erkennbar sein. Fehlende Disabled-/Error-Rueckmeldung
  gezielt ergaenzen, keine neuen Logs-Filter oder dekorative UIs erfinden.

## Dialog und Entscheidungen

1. Kimi las die Codebasis unabhaengig von diesem Report. Der erste
   Prozess erreichte sein Zeitlimit vor einer Abschlussantwort;
   seine Source-Beobachtungen wurden in derselben Sitzung weiterdiskutiert.
2. Codex lieferte Reproduktionen, 52 aktuelle ANSI-Captures und
   Raster-Captures; Kimi pruefte vier zentrale Captures und gezielte
   Code-Stellen, bestaetigte oder hinterfragte Befunde.
3. Codex korrigierte Severity und Formdimensionen, reproduzierte den
   Filter-Key-Bug und widersprach Border-Geometrie-Aenderungen.
   Kimi bestaetigte den finalen Konsens, nahm seine staerkere
   Manager-Maus-Severity sowie eine unbelegte Aussage zu Underline-
   Terminalverhalten zurueck.

**Gemeinsam beschlossen:** Verhalten und Layout zuerst; Erfolgspulses
sind Teil der Publish-Korrektur, keine spaetere Animation-Spielerei.
Keine Hover-Farbhighlights, keine r/rw-Chip-Praefixe, keine reservierte
Animationsspalte, keine neue Chip-Border-Geometrie und kein pauschaler
Background-Wechsel. Kontext-Hilfe bleibt zweizeilig oben. MQTT-Arbeitsflaeche
und explizite/Fallback-Publish-Semantik bleiben erhalten.

**Offen:** Konkrete Chip-Kodierung erst nach Vergleich der zwei Varianten.
Broker-Basic/Advanced-Gruppierung, neue Animationen und breites Redesign
sind zurueckgestellt. Topics-Suche ist reproduziert; analoge Key-Routing-
Risiken in Payloads/Traces sind noch gezielt zu testen.
Ctrl+D als globales Quit, auch aus Logs, ist bestehende Vorgabe, kein Bug.
Fuer Help/Logs keine neuen Defekte aus fehlenden Features abgeleitet.

## Verifikation und Grenzen

- `make test`: `go vet ./...` und `go test ./...` vollstaendig gruen.
- Dependabot-/CI-YAML strukturiert geprueft; `git diff --check` gruen.
- Temporaere Probe-Tests erfolgreich ausgefuehrt und wieder entfernt;
  keine Produktions-Go-Datei geaendert.
- 52 Render-Captures aus aktuellen Model-Views fuer vier Groessen,
  mit Offline-Fixtures, keine privaten Brokerdaten.
- Rasterisierung ueber pyte und JetBrainsMono/Pillow, feste dunkle
  Terminalfarben; keine Live-Terminal-/Theme- oder Animation-Zeitmessung.
  Kimi sichtpruefte vier Captures, Codex weitere zentrale Ansichten.
- Trace-Captures nutzten eine leere Liste und ein neues Formular;
  reale gefuellte/aktive Traces und Licht-/No-color-Terminals bleiben
  Akzeptanztests fuer die Umsetzung. Manual-Keyring-Beispiel nicht ausgefuehrt.
