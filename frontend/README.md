# New Pay – Frontend

React-basiertes Frontend für die New-Pay-Plattform (Gehaltseinschätzung & Peer Review).

## Tech Stack

- **Framework**: React 19, TypeScript 6
- **UI**: Mantine 9, Tabler Icons
- **Routing**: React Router 7
- **Build**: Vite
- **Linting/Formatting**: ESLint, Prettier

## Voraussetzungen

- Node.js 20+
- npm 10+
- Laufendes Backend (siehe `../backend/README.md`)

## Installation & Start

```bash
npm install
npm run dev
```

Die Anwendung ist standardmäßig unter `http://localhost:3001` erreichbar.

## Verfügbare Skripte

| Skript                | Beschreibung                       |
|-----------------------|------------------------------------|
| `npm run dev`         | Entwicklungsserver starten         |
| `npm run build`       | Produktions-Build erstellen        |
| `npm run preview`     | Produktions-Build lokal vorschauen |
| `npm run lint`        | ESLint ausführen                   |
| `npm run lint:fix`    | ESLint-Fehler automatisch beheben  |
| `npm run format`      | Code mit Prettier formatieren      |
| `npm run update:deps` | Alle Abhängigkeiten aktualisieren  |

## Projektstruktur

```
frontend/src/
├── components/     # Wiederverwendbare UI-Komponenten
├── contexts/       # React Contexts (z. B. Auth)
├── hooks/          # Custom Hooks
├── pages/          # Seitenkomponenten (Routing-Ziele)
│   ├── admin/      # Admin-Bereich
│   ├── auth/       # Login, Registrierung, Passwort-Reset
│   ├── profile/    # Nutzerprofil
│   ├── review/     # Peer-Review-Workflow
│   └── self-assessments/ # Selbsteinschätzungs-Workflow
├── services/       # API-Kommunikation mit dem Backend
├── types/          # TypeScript-Typdefinitionen
└── utils/          # Hilfsfunktionen
```

## Abhängigkeiten aktualisieren

```bash
npm run update:deps

## oder manuell mit npm-check-updates
npx npm-check-updates -u
npm install
```

## Lizenz

MIT
