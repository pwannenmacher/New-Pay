# Verschlüsselung und Key Management

## Überblick

Die Plattform nutzt eine lokale, anwendungsseitige Verschluesselungsarchitektur ohne externen Schluessel-Dienst.
Sensible Inhalte (z. B. Begruendungen) werden mit AES-256-GCM verschluesselt und zusaetzlich signiert.

- **Root-Key**: `ENCRYPTION_MASTER_KEY` (32 Bytes, hex-encodiert)
- **Key-Hierarchie**: Master-Key -> User-Key / Process-Key -> DEK
- **Signaturen**: Ed25519
- **Audit-Trail**: Hash-Chain (SHA-256)

## Architektur

```plain
ENCRYPTION_MASTER_KEY (Env, nicht in DB)
  |
  +-- User Key (Ed25519)          -> private keys verschluesselt in user_keys
  +-- Process Key (32 Byte)       -> verschluesselt in process_keys
         |
         +-- DEK (HKDF-SHA256, 32 Byte) pro Context

SecureStore:
  - AES-256-GCM fuer Daten
  - Ed25519 Signatur pro Record
  - Hash-Chain pro Prozess
```

## Key-Hierarchie

### 1) Master-Key

- Quelle: `ENCRYPTION_MASTER_KEY`
- Format: 64 Hex-Zeichen (32 Bytes)
- Zweck: Schutz von User-Private-Keys und Process-Keys
- Wichtig: Key-Verlust macht verschluesselte Daten unlesbar

### 2) User-Keys

- Typ: Ed25519 Keypair
- Speicherung:
  - Public Key: im Klartext in `user_keys.public_key`
  - Private Key: verschluesselt in `user_keys.encrypted_private_key`
- Nutzung: Signieren/Verifizieren von Datensaetzen

### 3) Process-Keys

- Typ: 32-Byte symmetrischer Schluessel
- Speicherung: verschluesselt in `process_keys.encrypted_key_material`
- Nutzung: Isolierung je fachlichem Prozess (z. B. Self-Assessment)

### 4) Data Encryption Key (DEK)

- Ableitung: HKDF-SHA256 (32 Bytes)
- Kontext: process-id + user-id + Schluesselmaterial
- Persistenz: nicht gespeichert, bei Bedarf neu abgeleitet

## Datenbankobjekte

### `user_keys`

- Public Key (hex)
- verschluesselter Private Key
- key_version

### `process_keys`

- verschluesseltes Key-Material
- key_hash (SHA-256)
- created_at / expires_at

### `encrypted_records`

- encrypted_data + nonce + tag
- Signaturdaten (Ed25519)
- Hash-Chain-Felder (`prev_record_hash`, `chain_hash`)
- Metadaten (`record_type`, `status`, `system_key_id`)

## Verschluesselungsablauf

### Daten speichern

1. Prozess- und User-Key laden
2. DEK mit HKDF-SHA256 ableiten
3. Payload mit AES-256-GCM verschluesseln
4. Ciphertext signieren (Ed25519)
5. Hash-Chain berechnen und Record speichern

### Daten lesen

1. Signatur validieren
2. DEK mit identischem Kontext ableiten
3. AES-256-GCM entschluesseln

## Sicherheitsmerkmale

- **AEAD (AES-256-GCM)**: Vertraulichkeit + Integritaet
- **Ed25519-Signatur**: Authentizitaet/Nachvollziehbarkeit
- **Hash-Chain**: Manipulationen erkennbar
- **Append-Only**: Historie unveraenderlich
- **Key-Separation**: Trennung zwischen Root-, User- und Process-Ebene

## Konfiguration

Pflichtvariable in `.env`:

```bash
ENCRYPTION_MASTER_KEY=<64-hex-chars>
```

Key erzeugen:

```bash
openssl rand -hex 32
```

## Betrieb und Recovery

- `ENCRYPTION_MASTER_KEY` immer getrennt vom Quellcode sichern (Password Manager / Secret Store)
- niemals in Git committen
- Schluesselmaterial in Backups ebenfalls als Secret behandeln

## Troubleshooting

### Fehler: `ENCRYPTION_MASTER_KEY is required`

- Variable fehlt in `.env`
- Startparameter/Container-Env pruefen

### Fehler: `invalid ENCRYPTION_MASTER_KEY`

- Key ist nicht hex-encodiert oder hat nicht 64 Zeichen

### Fehler: `failed to derive data encryption key`

- inkonsistenter Kontext (process/user) oder defektes Key-Material
- Process-Key und User-Key-Eintraege pruefen

## Referenzen

- [AES-GCM (NIST SP 800-38D)](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf)
- [HKDF (RFC 5869)](https://datatracker.ietf.org/doc/html/rfc5869)
- [Ed25519 (RFC 8032)](https://datatracker.ietf.org/doc/html/rfc8032)
- [Go: crypto/cipher](https://pkg.go.dev/crypto/cipher)
- [Go: crypto/ed25519](https://pkg.go.dev/crypto/ed25519)
