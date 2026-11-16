# NELKO P21 – ASK Edition v1.0.1: Windows 11-afprøvning

## Hent programmet

1. Åbn GitHub → Actions → **ASK Edition - Windows x64**.
2. Vælg en grøn kørsel på `develop/ask-edition-v1.0.1`.
3. Hent artefaktet `NELKO-P21-ASK-Edition-v1.0.1-windows-amd64` (GitHub-login kan være nødvendigt).
4. Udpak downloadens ZIP-fil og derefter programpakken. Start EXE-filen.

Programmet er ikke kodesigneret. Ingen Go-installation er nødvendig for at køre det.

## Fysisk printertest

- Par P21 i Windows' Bluetooth-indstillinger. Brug den eksisterende udgående COM-port.
- Vælg printeren under Bluetooth eller porten under **Avanceret**, og tryk **Tilslut**.
- Bekræft navn og version i titel, statuslinje og **Hjælp → Om programmet**.
- Vælg 14x40 mm og sværtningsgrad 10. Udskriv `ASK – ÆØÅ æøå 123` i Tekst.
- Prøv Go Regular, Go Bold og Go Mono, begge tekstretninger og flere skriftstørrelser.
- Åbn Designer, tilføj Tekst og vælg tekstobjektet. Redigér flere tegn og linjer i feltet under lærredet. Tastaturfokus og markering skal blive i tekstfeltet.
- Flyt, skalér og rotér objektet. Udskriv og sammenlign med forhåndsvisningen.
- Gem etiketten i Bibliotek med et navn. Åbn til redigering, ret indholdet, gå tilbage til Bibliotek og vælg **Erstat med aktuel etiket**.
- Omdøb etiketten, markér den som skabelon og prøv at slette en kopi. De to medfølgende skabeloner kan også redigeres og slettes.
- Luk programmet og start igen. Kontrollér tekst, skrifttype, skriftstørrelse, etiketstørrelse, sværtningsgrad, bibliotek og sidste printer. Med **Tilslut automatisk igen** slået til skal forbindelsen åbnes igen.
- Start programmet, mens printeren er slukket, og tænd den senere. Der forsøges gentilslutning hvert femte sekund efter en forbindelsesfejl.
- Sluk og tænd printeren under brug. Kontroller gentilslutning efter registreret forbindelsestab eller skrivefejl. Et mislykket udskriftsjob gensendes aldrig automatisk; kontrollér etiketten og tryk Udskriv igen.
- Tryk **Afbryd**, og kontrollér, at forbindelsen forbliver afbrudt resten af sessionen. Fravælg automatisk gentilslutning for også at deaktivere den ved næste start.
- Skift mellem **Language / Sprog → English / Dansk**. Indhold og indstillinger skal bevares.
- Åbn **Diagnostik → Test batteri**. Hvis printerens eksisterende `BATTERY?`-forespørgsel ikke giver et gyldigt niveau, vises **Batteri: ukendt**.

## Forbindelsesdetektion

Bluetooth- og TSPL-implementeringen er bevaret. På Windows kan COM-porten stadig eksistere, selv om printeren er slukket. Hvis **Automatiske batteriforespørgsler** er slået til, og printeren tidligere har svaret, registreres tre manglende svar som forbindelsestab. Hvis dens firmware ikke understøtter forespørgslen, udløses gentilslutning af en fejl ved åbning eller udskrivning; tilstedeværelsen af en COM-port alene beviser ikke en aktiv Bluetooth-forbindelse.

## Lokale data

Indstillinger og bibliotek gemmes via Fyne under app-id `dk.ask.nelko-p21`. På Windows ligger præferencerne normalt i `%APPDATA%\fyne\dk.ask.nelko-p21\preferences.json`. Sikkerhedskopiér hele mappen, mens programmet er lukket. Biblioteket indeholder indlejrede billeder i Designer-etiketter. Etiketter kan også gemmes separat som `.nlbl` via Designers **Gem**-knap.

Automatiske tests kan ikke bekræfte Bluetooth-forbindelsen, det fysiske udskriftsresultat eller batterisvaret fra din P21. Notér Windows-version, COM-port, labeltype og det konkrete trin ved fejl.


## Ny diagnostisk reference i v1.0.1

1. Luk PowerShell-forbindelsen og andre programmer, der holder COM5 åben. Tilslut COM5 i ASK Edition (115200 baud, 8N1).
2. Lad **Diagnostik → Automatiske batteriforespørgsler** være slået fra (standard). Der sendes dermed ingen batteriforespørgsel ved tilslutning eller under sundhedskontrol. Andre etiketindstillinger bevares.
3. Tryk **Reference-testudskrift**. Testen bruger altid 14x40 mm, density 15, bitmap mode 1 og én kopi, uanset den aktuelle etiket og tekst. Den sender `1B 21 6F`, venter 500 ms, sender kommandolinjer/header/binære data i rækkefølge og venter 3 sekunder efter `PRINT 1`.
4. Kontrollér det fysiske resultat. Testmønstret indeholder en ramme og tre bjælker i 3408 rå bytes. Bitmapindholdet er nyt, da PowerShell-testens bitmapbytes ikke er leveret. Notér også, hvis mønstret er inverteret.
5. Tryk **Gem kommunikationslog** straks efter testen. Loggen indeholder tidsstempler med tidszone, præcise TX/RX-bytes i ASCII og HEX samt WAIT-hændelser. Succes i brugerfladen bekræfter kun skrivningen og ventetiden, ikke en fysisk udskrift.
6. Test derefter almindelig udskrivning på 14x40 mm og density 15. Normale jobs bruger nu også 500 ms og 3 sekunders ventetid. De sendes som et samlet job; referenceknappen skriver kommandolinjer, bitmap-header, payload, CR+LF og PRINT separat. Sammenlign log og papirresultat.
7. Kør separat **Cancel Pause** og **Test batteri**, og gem endnu en log. Cancel Pause sender kun de tre bytes og venter 500 ms. Batteritesten bruger den hidtidige `BATTERY?`-kommando. Manglende/ukendt svar er ikke bevis for en defekt printer og udløser ikke automatisk genudskrivning.

Loggen gemmes kun i hukommelsen, indtil du selv eksporterer den; den kan indeholde etikettens tekst og billeddata. Den nyeste log er begrænset til cirka 256 KiB formateret tekst, og det markeres, hvis ældre poster er faldet ud. Gem efter hver test. ASCII bruger escape-sekvenser for binære tegn; HEX viser de faktiske byteværdier. RX vises, når programmet læser et kommandosvar; der er ingen separat læsetråd, der kan stjæle svaret.

Se også [sammenligning med PowerShell-sekvensen](PRINT-DIAGNOSTICS-v1.0.1.md). Det er endnu ikke fastslået, hvilken forskel der har betydning på fysisk hardware.
