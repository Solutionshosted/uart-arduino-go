# arduino

Lokales Go-Paket für die gesamte serielle Kommunikation mit dem Arduino.
Es gehört zum Go-Modul `reactorcoolergame`; eine separate Installation ist
nicht nötig.

```go
import "reactorcoolergame/arduino"

arduino.Run(ctx, func(frame arduino.Frame) {
    // frame.Type enthält z. B. EVALUATION.
    // frame.Fields enthält ampel und solved als Zeichenketten.
})
```

`Run` erkennt den USB-Port unter Linux oder macOS automatisch und öffnet ihn
mit 115200 Baud. Nach Verbindungsverlust versucht es alle zwei Sekunden eine
neue Verbindung. USB-Geräte erhalten beim Öffnen zwei Sekunden Startzeit.
Ein abgebrochener Kontext beendet die Schleife und schließt den Port.

Das Paket setzt Teilstücke zu vollständigen Zeilen zusammen, unterstützt CRLF
und ignoriert ungültige oder über 4096 Bytes lange Zeilen. `ParseFrame` ist
auch separat verfügbar. Feldnamen werden in Kleinbuchstaben zurückgegeben.

Der Handler wird synchron für jede vollständige Nachricht aufgerufen.
Die Verarbeitung von Teams, Bewertungen, MQTT und MongoDB bleibt außerhalb
dieses Pakets. Es gibt keine Kommandozeilenoption für den Port.

Tests aus `ReactorCoolerCode/rpi`:

```sh
go test -race ./...
```
