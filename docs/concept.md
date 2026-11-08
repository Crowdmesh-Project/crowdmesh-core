# AviaMesh — Dokumentacja Koncepcyjna i Architektoniczna

> **Wersja dokumentu:** 0.1.0-draft  
> **Status:** Zaakceptowany do implementacji  
> **Język wdrożenia:** Go (1.22+)  
> **Cel:** Deterministyczny symulator oraz osadzalny rdzeń sieci ad-hoc/mesh w środowisku pozbawionym infrastruktury (kabina pasażerska samolotu).

---

## 1. Wprowadzenie i Teza Projektowa

### 1.1. Problem
Współczesne podróże lotnicze charakteryzują się niemal kompletnym odcięciem pasażerów od taniej lub darmowej łączności internetowej w trakcie lotu (Wi-Fi pokładowe jest drogie, wolne lub niedostępne). Jednocześnie w przestrzeni kabiny o wymiarach ok. $25 \times 4\text{ m}$ przebywa od 150 do 250 osób wyposażonych w urządzenia z interfejsem Bluetooth Low Energy (BLE) oraz Wi-Fi Aware (NAN).

Tradycyjne protokoły sieciowe (np. mesh IP w standardzie 802.11s czy klasyczny routing odległościowy typu AODV/OLSR) zawodzą w tym środowisku z powodu:
1. **Wysokiej gęstości węzłów:** syndrom burzy rozgłoszeniowej (*Broadcast Storm Problem*), kolizje w pasmach ISM 2.4 GHz.
2. **Tłumienia przez ciała ludzkie:** sygnał radiowy 2.4 GHz jest silnie tłumiony przez wodę (tkanki pasażerów absorbują od 3 dB do 10 dB na osobę/rząd).
3. **Braku centralnego koordynatora:** brak dostępu do serwera NTP, stacji bazowej czy stabilnych połączeń punkt-punkt.
4. **Ograniczeń systemów operacyjnych:** restrykcyjne zarządzanie energią na iOS i Androidzie uniemożliwia utrzymywanie ciągłych połączeń GATT w tle.

### 1.2. Teza
Możliwe jest zaprojektowanie lekkiego, bezpołączeniowego protokołu komunikacyjnego opartego na rozgłaszaniu pakietów bezpołączeniowych (BLE Advertising / Wi-Fi NAN Datagrams), który:
- Zapewnia **asynchroniczną dostarczalność wiadomości tekstowych** między dowolnymi fotelami w samolocie bez centralnego serwera.
- Ogranicza nadmiarowe retransmisje o minimum **60–80%** w stosunku do naiwnego zalewania (flooding) za pomocą tłumienia licznikowego (*counter-based suppression*) z adaptacyjnym jitterem.
- Utrzymuje spójność ostateczną (*eventual consistency*) bazy wiadomości dzięki strukturze CRDT (G-Set) uporządkowanej przez Hybrid Logical Clocks (HLC).
- Jest weryfikowalny w **deterministycznym, dyskretnym symulatorze**, którego wyniki są w 100% powtarzalne dla ustalonego ziarna losowości (*seed*).

---

## 2. Kluczowe Zasady Projektowe

1. **Rdzeń bez I/O (Pure State Machine):**
   Rdzeń protokołu (`core`) jest czystym automatem stanowym. Nie wykonuje wywołań systemowych, nie sprawdza czasu (`time.Now()`), nie zarządza socketami, nie alokuje wątków ani nie korzysta z dysku. Na wejściu przyjmuje zdarzenie (`Event`), na wyjściu zwraca listę akcji (`Action`).
2. **Determinizm Ponad Wszystko:**
   Każdy proces losowy w symulatorze opiera się na jawnie przekazanym generatorze pseudolosowym zainicjalizowanym stałym seedem. Pętla symulacji pojedynczego przebiegu jest jednowątkowa.
3. **Gotowa i Sprawdzona Kryptografia:**
   Użycie wyłącznie zweryfikowanych standardów: X25519, ChaCha20-Poly1305 / NaCl Box (`golang.org/x/crypto/nacl/box`), HMAC-SHA256. Zero własnoręcznie tworzonych algorytmów kryptograficznych.
4. **Oddzielenie Modelu od Pomiarów:**
   Symulator opiera się na jawnych, parametryzowalnych równaniach propagacji fal i kolizji. Wszelkie liczby w dokumentacji reprezentują wyniki modelu, nie obietnice sprzętowe.

---

## 3. Architektura Systemu i Granica Rdzenia

```
               +-------------------------------------------------+
               |              Aplikacja Mobilna                  |
               |         (Android / iOS / UI / BLE Stack)        |
               +-------------------------------------------------+
                                       |
                   [Event: BLE Advert Rx, Timer, UserSend]
                                       |
                                       v
               +-------------------------------------------------+
               |             AviaMesh Core Engine                |
               |       (Czysty automat stanowy w Go)             |
               |                                                 |
               |  - Parser Ramek Binarnych (15 B Header)        |
               |  - Strategia Forwardingu (Trickle / Gossip)    |
               |  - Cache Deduplikacji (LRU)                    |
               |  - Magazyn CRDT (G-Set + HLC)                  |
               |  - Anti-Entropy & Beaconing                     |
               +-------------------------------------------------+
                                       ^
                   [Event: Virtual Radio Packet, SimTimer]
                                       |
               +-------------------------------------------------+
               |            Deterministyczny Symulator           |
               |         (Model kabiny, fal radiowych, RNG)      |
               +-------------------------------------------------+
```

### 3.1. Automat Stanowy: `Node.Handle(Event) -> []Action`

Interfejs węzła definiuje funkcja przejścia:
$$\mathcal{S}_{t+1}, \mathcal{A} = f(\mathcal{S}_t, \mathcal{E})$$

#### Zdarzenia wejściowe (`Event`):
- `EventFrameReceived(Data []byte, RSSI int, LinkType)` – odebranie surowej ramki z medium radiowego.
- `EventTimerExpired(TimerID uint64)` – wygaśnięcie wirtualnego timera (np. jitter retransmisji, interwał beaconingu).
- `EventUserSend(Payload []byte, RecipientPubKey [32]byte)` – użytkownik zlecił wysłanie nowej wiadomości.
- `EventNeighborDiscovered(NodeTag uint32, RSSI int)` – wykrycie obecności sąsiada (np. z beaconu).
- `EventNeighborLost(NodeTag uint32)` – brak sygnału od sąsiada przez określony czas.

#### Akcje wyjściowe (`Action`):
- `ActionTransmit(Data []byte, LinkType, Priority)` – żądanie emisji ramki w medium radiowym.
- `ActionSetTimer(TimerID uint64, DurationMs uint64)` – zaplanowanie timera w schedulerze wirtualnym / systemowym.
- `ActionCancelTimer(TimerID uint64)` – anulowanie aktywnego timera (np. w tłumieniu licznikowym Trickle).
- `ActionDeliverToUser(MessageID [8]byte, Plaintext []byte, HLC Timestamp)` – przekazanie odszyfrowanej wiadomości do UI.
- `ActionPersistState(Key string, Data []byte)` – zrzut stanu do trwałego magazynu (opcjonalnie).

---

## 4. Model Fizyczny Kabiny i Propagacji Radiowej

### 4.1. Geometria Kabiny
Standardowy samolot wąskokadłubowy (np. Airbus A320 / Boeing 737):
- **Konfiguracja:** 30 rzędów foteli w układzie 3–3 (fotele A, B, C | alejka | D, E, F).
- **Liczba miejsc:** 180 pasażerów.
- **Wymiary:**
  - Odstęp między rzędami (*seat pitch*): $\Delta x = 0.8\text{ m}$.
  - Odstęp między fotelami w rzędzie: $\Delta y = 0.5\text{ m}$.
  - Szerokość alejki: $\Delta y_{\text{aisle}} = 0.8\text{ m}$.
  - Całkowita długość kabiny: $L \approx 24\text{ m}$, szerokość: $W \approx 3.8\text{ m}$.

Każdy fotel posiada współrzędne kartezjańskie $(x_i, y_i, z_i)$ w metrach.

### 4.2. Model Propagacji Fal (Radio Path Loss)
Moc odbieranego sygnału $P_{rx}$ w odległości $d$ od nadawcy jest wyliczana na podstawie modelu logarytmiczno-odległościowego z cieniowaniem log-normalnym (*Log-Distance Path Loss with Shadowing*):

$$PL(d) = PL(d_0) + 10 \cdot n \cdot \log_{10}\left(\frac{d}{d_0}\right) + X_\sigma + \sum A_{\text{human}}$$

Gdzie:
- $d_0 = 1.0\text{ m}$ (dystans referencyjny),
- $PL(d_0) \approx 40\text{ dB}$ (tłumienie w wolnej przestrzeni dla $2.4\text{ GHz}$ na dystansie $1\text{ m}$),
- $n = 2.4 \dots 3.2$ (współczynnik tłumienia środowiska zamkniętego z przeszkodami metalowymi),
- $X_\sigma \sim \mathcal{N}(0, \sigma^2)$ z $\sigma = 3.0\text{ dB}$ (losowe cieniowanie wywołane konstrukcją kadłuba),
- $\sum A_{\text{human}}$ — skumulowane tłumienie na linii wzroku (LOS). Każde ciało pasażera przecinające odcinek między nadawcą a odbiorcą wprowadza dodatkowe tłumienie $A_{\text{passenger}} \in [2.5, 4.0]\text{ dB}$.

### 4.3. Model Odbioru i Kolizji (SINR)
Moc odbierana:
$$P_{rx}\text{ [dBm]} = P_{tx}\text{ [dBm]} - PL(d)$$

1. **Czułość odbiornika (Sensitivity Threshold):** Jeśli $P_{rx} < P_{\text{sens}}$ (np. $-93\text{ dBm}$), ramka nie zostaje wykryta przez odbiornik.
2. **Kolizje pakietów:** Jeśli w oknie czasowym transmisji pakietu $A$ do odbiorcy $R$ dociera inna transmisja $B$, wyliczany jest stosunek sygnału do zakłóceń plus szum (*SINR*):
   $$\text{SINR} = \frac{P_{rx, A}}{P_{\text{noise}} + \sum P_{rx, \text{interferers}}}$$
   Pakiet $A$ zostaje poprawnie zdekodowany tylko wtedy, gdy $\text{SINR} \ge \gamma_{\text{capture}}$ (efekt przejęcia, *capture effect*, typowo $3\text{ dB}$ dla demodulatorów GFSK). W przeciwnym wypadku oba pakiety ulegają zniszczeniu (kolizja).

---

## 5. Protokół Binarny i Format Ramki

Struktura ramki jest zoptymalizowana pod minimalne dopuszczalne MTU rozgłoszeń BLE (ang. *legacy advertising data*, limit 31 B, z czego realnie ok. 24 B na payload po odliczeniu flag BLE, lub *BLE Extended Advertising* z MTU konfigurowalnym w zakresie 100–240 B).

### 5.1. Binarny Nagłówek Ramki (15 Bajtów)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Ver  | Type  |      TTL      |          Message ID           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|                    Message ID (ciąg dalszy)                   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       Message ID (cd.)        |         Recipient Tag         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               +
|     Recipient Tag (cd.)       |  Payload Len  |    Payload    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+               |
|                           ...                                 |
```

| Przesunięcie (B) | Pole | Rozmiar | Typ | Opis |
|---|---|---|---|---|
| 0.0 | **Version** | 4 bity | `uint8` (high nibble) | Wersja protokołu (aktualnie `0x1`). |
| 0.4 | **Frame Type** | 4 bity | `uint8` (low nibble) | `0x0` = Data, `0x1` = Beacon, `0x2` = SyncRequest, `0x3` = SyncResponse. |
| 1.0 | **TTL** | 1 bajt | `uint8` | Hop limit (np. startowo 5–7). Zmniejszany o 1 przy każdym przekazaniu. |
| 2.0 | **Message ID** | 8 bajtów | `[8]byte` | Pseudolosowy identyfikator wiadomości (do deduplikacji). |
| 10.0 | **Recipient Tag** | 4 bajty | `[4]byte` | Rotujący znacznik adresata (patrz sekcja 5.2). `0xFFFFFFFF` = Broadcast. |
| 14.0 | **Payload Length** | 1 bajt | `uint8` | Liczba bajtów ładunku $L$ ($0 \le L \le \text{MTU} - 15$). |
| 15.0 | **Payload** | $L$ bajtów | `[]byte` | Szyfrowana zawartość lub dane synchronizacji. |

### 5.2. Rotujący Recipient Tag (Prywatność i Ochrona Metadanych)
Zamiast jawnego klucza publicznego odbiorcy, w nagłówku umieszcza się 32-bitowy znacznik:
$$\text{RecipientTag} = \text{Truncate}_{32}\left(\text{HMAC-SHA256}(K_{\text{contact}}, \text{Epoch})\right)$$
- $\text{Epoch} = \lfloor \text{UnixTime} / 600 \rfloor$ (okno 10-minutowe).
- Odbiorca sprawdza pasowanie tagu dla $\text{Epoch}-1$, $\text{Epoch}$ oraz $\text{Epoch}+1$ (odporność na desynchronizację zegarów).
- Węzły pośredniczące widzą losowy ciąg bitów zmieniający się co 10 minut.

### 5.3. Ładunek Szyfrowany (`Payload`)
Struktura danych po odszyfrowaniu za pomocą kluczy X25519 / ChaCha20-Poly1305:
1. **Nonce:** 24 bajty (przesyłany jawnie na początku payloadu lub wyprowadzony deterministycznie z Message ID + Sender ID).
2. **HLC Timestamp:** 8 bajtów (uporządkowanie w CRDT).
3. **Sender Public Key:** 32 bajty (do weryfikacji tożsamości nadawcy).
4. **Treść tekstowa:** zmienna długość (np. UTF-8).
5. **Poly1305 Auth Tag:** 16 bajtów (integralność i uwierzytelnienie).

---

## 6. Algorytmy Forwardingu i Kontrola Ruchu

### 6.1. Flooding z Deduplikacją (Baseline)
- Każdy węzeł posiada filtr LRU przechowujący ostatnie $N=1000$ identyfikatorów `MessageID`.
- Po odebraniu ramki:
  - Jeśli `MessageID` jest w cache $\to$ odrzuć.
  - Jeśli `TTL <= 1` $\to$ odrzuć.
  - W przeciwnym razie: dodaj do cache, zmniejsz `TTL` o 1 i natychmiast zaplanuj nadanie z minimalnym losowym jitterem ($10 \dots 30\text{ ms}$).

### 6.2. Gossip Probabilistyczny
- Węzeł retransmituje ramkę z prawdopodobieństwem $p \in (0, 1]$ (np. $p = 0.6$).
- Zmniejsza liczbę nadawań w centrum gęstej sieci, lecz zwiększa ryzyko niedostarczenia wiadomości na krańcach kabiny przy niskiej penetracji aplikacji.

### 6.3. Tłumienie Licznikowe (Adaptacyjny Algorytm Typu Trickle)
Mechanizm inspirowany RFC 6206:
1. Po odebraniu nowej ramki, węzeł losuje czas oczekiwania $t \in [t_{\min}, t_{\max}]$ (np. $50 \dots 200\text{ ms}$) i inicjalizuje licznik $c = 1$.
2. W trakcie oczekiwania węzeł nasłuchuje medium:
   - Jeśli usłyszy retransmisję ramki o tym samym `MessageID` od innego węzła: $c \leftarrow c + 1$.
3. Po upływie czasu $t$:
   - Jeśli $c \le k$ (gdzie $k$ to próg redundancji, np. $k = 1$ lub $k = 2$): **węzeł nadaje ramkę**.
   - Jeśli $c > k$: **węzeł tłumi (anuluje) nadanie**, uznając, że sąsiedzi skutecznie pokryli obszar retransmisją.

---

## 7. Spójność Danych i Synchronizacja (DTN & Anti-Entropy)

W przypadku rozłączenia sieci (np. brak pasażerów z aplikacją w rzędach 10–15) kabina dzieli się na odizolowane wyspy.

### 7.1. Hybrydowy Zegar Logiczny (HLC)
Zegary smartfonów pasażerów w trybie samolotowym mogą różnić się o sekundy lub minuty. Aby zapewnić bezkolizyjne i monotoniczne porządkowanie wiadomości, stosowany jest zegar HLC (Kulkarni et al., 2014):
$$\text{HLC} = \langle l, c \rangle$$
- $l$ — fizyczny czas w milisekundach (zabezpieczony przed cofaniem się),
- $c$ — 16-bitowy licznik logiczny dla zdarzeń o tym samym $l$.

Relacja porządku:
$$\langle l_1, c_1 \rangle < \langle l_2, c_2 \rangle \iff (l_1 < l_2) \lor (l_1 = l_2 \land c_1 < c_2)$$

### 7.2. Magazyn CRDT (G-Set)
Zbiór wiadomości jest strukturą typu Grow-only Set (G-Set). Wiadomości mogą być wyłącznie dopisywane. Scalanie zbiorów jest przemienne, łączne i idempotentne:
$$\mathcal{S}_A \cup \mathcal{S}_B = \mathcal{S}_{\text{merged}}$$

### 7.3. Beaconing i Anti-Entropy
Każdy węzeł okresowo (np. co $1.5\text{ s}$) emituje kompaktowy beacon:
- **Stan zbioru (Digest):**
  $$\text{Digest} = \left(\text{Count} \ll 32\right) \oplus \bigoplus_{i=1}^{M} \text{Hash32}(\text{MessageID}_i)$$
  gdzie uwzględniane są wiadomości z ostatnich 30 minut.
- Gdy dwa węzły znajdą się w zasięgu i wykryją różnicę w Digest:
  1. Węzeł inicjujący wysyła `SyncRequest` z listą swoich ostatnich identyfikatorów.
  2. Węzeł odpowiadający zwraca brakujące wiadomości w ramkach danych.
- **Wędrowiec (Store-and-Forward):** Pasażer idący do toalety odbiera wiadomości z przodu samolotu, przemieszcza się fizycznie przez przerwę w łączności i przekazuje je do wyspy na tyle kabiny drogą anti-entropy.

---

## 8. Model Zagrożeń (Threat Model) i Założenia Bezpieczeństwa

### 8.1. Zapewniane Własności
- **Poufność treści (Confidentiality):** Tylko właściciel docelowego klucz prywatnego X25519 może odczytać zawartość `Payload`.
- **Integralność i uwierzytelnienie:** Modyfikacja szyfrogramu skutkuje natychmiastowym odrzuceniem przez Poly1305.
- **Ochrona przed inwigilacją pasywną na dystansie:** Zewnętrzny podsłuchiwacz nie widzi stałych identyfikatorów użytkowników (dzięki rotującym tytule Recipient Tag).

### 8.2. Świadome Kompromisy i Ograniczenia (Non-Goals)
1. **Korelacja wewnątrz epoki:** W obrębie jednego 10-minutowego okna czasowego wszystkie wiadomości do tego samego odbiorcy mają identyczny `RecipientTag`. Obserwator może powiązać ze sobą te wiadomości.
2. **Brak Perfect Forward Secrecy (PFS):** Ze względu na asynchroniczny charakter sieci i store-and-forward (odbiorca może być offline przez 20 minut), nie jest stosowany interaktywny protokół Signal/Double Ratchet. Przejęcie klucza prywatnego pozwala na odszyfrowanie wcześniejszych przechwyconych wiadomości.
3. **Ataki Sybil i zalewanie (Flood DoS):** Złośliwy węzeł generujący miliony losowych ramek może wysycić pasmo radiowe. Mitygacja opiera się na lokalnym rate-limitingu na poziomie odbiornika.

---

## 9. Plan Eksperymentów Symulacyjnych

Symulator przeprowadza minimum 30 powtórzeń (różne ziarna losowości RNG) dla każdego scenariusza badawczego. Wyniki raportują medianę oraz 95% przedział ufności (bootstrap CI).

```
+-----+------------------------------------------+-----------------------------------------------------+
| Kod | Hipoteza Badawcza                        | Główne Metryki                                      |
+-----+------------------------------------------+-----------------------------------------------------+
| E1  | Baseline: Ograniczenia czystego floodingu| PDR (Packet Delivery Ratio), kolizje, TX/msg, delay |
| E2  | Zysk z algorytmu Trickle (k=1..3) vs PDR | Liczba nadawań na odebrany pakiet vs opóźnienie     |
| E3  | Granica perkolacji sieci (adopcja 10-90%)| Krytyczny próg penetracji kabiny dla spójności     |
| E4  | Skuteczność wędrowca w scalaniu wysp     | Czas pełnej konwergencji G-Set w obu partycjach    |
| E5  | Koszt energetyczny beaconingu            | Liczba bajtów i nadawań na węzeł na minutę         |
+-----+------------------------------------------+-----------------------------------------------------+
```

---

## 10. Implementacja w Go: Standardy Inżynieryjne

1. **Brak Globalnego Stanu:** Wszelkie struktury przyjmują zależności (`Rng`, `Clock`) w konstruktorach lub wywołaniach metod.
2. **Optymalizacja Alokacji:** Parser ramek binarnych nie tworzy nowych obiektów na stercie w pętli `Handle()` — operuje na wycinkach pamięci (`sub-slices`) buforów.
3. **Natywne Testy:**
   - `testing.F` (`go test -fuzz`) dla odporności parsera na błędy parsowania (malformed frames).
   - Testy własnościowe monotoniczności HLC i konwergencji G-Set za pomocą biblioteki `pgregory.net/rapid`.
   - Testy regresyjne determinizmu: asercja zgodności hashów stanu symulacji dla identycznego seeda.
