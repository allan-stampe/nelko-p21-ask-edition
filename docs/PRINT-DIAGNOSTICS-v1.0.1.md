# Sammenligning: PowerShell-reference og ASK Edition v1.0.1

Brugeren har fysisk bekræftet udskrivning fra PowerShell på Windows 11 via COM5, 115200 baud, 8N1. Begge tidligere programmer havde udskrivningsproblemer. Der er endnu ingen fysisk dokumentation for, hvilken forskel der løser problemet. v1.0.1 er en instrumenteret kandidat til afprøvning, ikke en bekræftet hardwareløsning.

| Punkt | ASK Edition v1.0.0 | Fysisk testet sekvens | v1.0.1 |
|---|---|---|---|
| Port og format | Valgt COM-port, 115200, 8N1 | COM5, 115200, 8N1 | Bevarer portvalg og format; COM5 vælges ved afprøvning |
| Cancel Pause | `1B 21 6F`, ingen CR/LF; fejl blev ignoreret | Samme tre bytes | Samme bytes; fejl stopper jobbet |
| Vent efter Cancel Pause | 100 ms | 500 ms | 500 ms, også på separat Cancel Pause-knap |
| Rækkefølge | SIZE, GAP, DIRECTION, DENSITY, CLS, BITMAP, PRINT | Samme | Samme |
| GAP-format | `GAP 5.0 mm,0.0 mm` | `GAP 5.0 mm,0 mm` | Nul-offset skrives præcis `0 mm`; betydningen forventes ens, men er ikke isoleret testet |
| Sværtningsgrad | Brugerindstilling, standard 10 | 15 | Almindelige jobs beholder brugerens valg; referencetest tvinger 15 |
| Bitmap mode | `1` | `1` | `1`; ingen ændring til 0 eller 2 |
| Bitmapmål | For 14x40 mm: 12 bytes × 284 rækker | 12 × 284 = 3408 bytes | Referencetest har præcis 3408 rå bytes |
| Binær framing | Header direkte efterfulgt af bitmap og derefter CR+LF | Samme | Samme; ingen tekstkodning eller CR+LF før bitmapdata |
| Bitmappayload | Afledt af tekst/billede; eksisterende inversion | Ikke oplyst | Almindelig konvertering/inversion bevares. Reference har et nyt, deterministisk mønster |
| Skrivningsgrænser | Hele jobbet i én Write efter Cancel Pause | Beskrevet som separate kommandoer/header/data | Normal jobgrænse bevares; reference bruger separate writes med samme samlede bytestream |
| Delvise writes | Antal skrevne bytes blev ignoreret | PowerShell testet fysisk | Skriver resten, stopper på fejl eller nul-byte-write |
| Vent efter PRINT | Ingen | 3 sekunder | 3 sekunder under samme forbindelseslås |
| Batteriforespørgsler | `BATTERY?` ved tilslutning og hvert femte sekund | Indgår ikke i reference | Manuel test; automatisk polling kan slås til i Diagnostik og huskes |
| Protokollog | Ingen | Ikke oplyst | Tidsstemplede API-TX/RX, ASCII, HEX, fejl, åbning/lukning og ventetider |

## Referenceknappens skrivninger

```text
TX: 1B 21 6F                  (tre binære bytes; ingen linjeafslutning)
WAIT: 500 ms
TX: SIZE 14.0 mm,40.0 mm\r\n
TX: GAP 5.0 mm,0 mm\r\n
TX: DIRECTION 0,0\r\n
TX: DENSITY 15\r\n
TX: CLS\r\n
TX: BITMAP 0,0,12,284,1,       (ingen CR/LF efter headeren)
TX: 3408 binære bitmapbytes
TX: \r\n
TX: PRINT 1\r\n
WAIT: 3 sekunder
```

Serial-API'et kan dele en skrivning i mindre bidder; de resterende bytes sendes i samme rækkefølge. Loggen viser det faktiske antal accepterede bytes ved hver kaldt Write. Ingen anden kommando kan indgå under de låste ventetider. Der ventes også, selv om alle bytes er accepteret af Windows' driver; det er ikke en kvittering fra printeren.

Der ændres ikke i platformenes Bluetooth-backends, COM-portdetektion eller baud/data/paritets/stopbitvalg. MIT-licensen og Tyler C-krediteringen bevares. `.nlbl`, bibliotek, sprog, Designer og øvrige indstillinger fortsætter med samme lagerformat.

## Batterisvar

`BATTERY?` bevares; den hidtidige binære fortolkning (`BATTERY` + én procentbyte) får validering af præfiks og interval. Binære bytes trimmes ikke længere som whitespace, og delvise svar eller svar uden LF bevares. Alle rå modtagne bytes logges, også ved fejl. Denne fortolkning er ikke bekræftet med jeres PowerShell-test; ukendte eller manglende svar vises som ukendt, ikke som 0 %.

## Automatiske kontroller

Tests sammenligner referencekommandoerne byte for byte, de ni jobdele, de 3408 payloadbytes, de tre Cancel Pause-bytes og begge pausers værdier og placering. De kontrollerer også korte writes, fejl ved Cancel Pause, udelukkelse af batterikommandoer under udskrivning, binære batterisvar, faktisk TX/RX-log og logeksport samt tidligere funktioner. Ventetiderne testes med en injiceret ventefunktion; fysisk Bluetooth-timing skal stadig efterprøves under Windows 11.
