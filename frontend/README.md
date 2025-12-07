# New Pay Frontend

Modern React frontend for the New Pay salary estimation and peer review platform.

## Tech Stack

- **React 19** - Latest React with improved performance
- **TypeScript 5.9** - Type-safe JavaScript
- **Vite** - Fast build tool and dev server
- **Mantine 7** - Comprehensive component library
- **React Router 7** - Client-side routing
- **Tabler Icons** - Icon library

## Features

### Authentication & Security
- ✅ JWT-based authentication with automatic token refresh
- ✅ Login and registration pages with form validation
- ✅ Email verification flow
- ✅ Password reset functionality
- ✅ OAuth 2.0 integration (Google, Facebook)
- ✅ Protected routes with role-based access control

### User Management
- ✅ User profile viewing and editing
- ✅ Admin dashboard for user management
- ✅ Role assignment interface
- ✅ Audit log viewer

### Design
- ✅ Responsive design (mobile and desktop)
- ✅ Full-width layout on desktop
- ✅ Mobile navigation drawer
- ✅ Dark/light mode support
- ✅ Modern, accessible UI components

## Quick Start

### 1. Install Dependencies

```bash
npm install
```

### 2. Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

### 3. Start Development Server

```bash
npm run dev
```

The application will be available at `http://localhost:5173`

### 4. Build for Production

```bash
npm run build
```

## Project Structure

```
src/
├── components/          # Reusable UI components
│   ├── auth/           # Authentication components
│   ├── layout/         # Layout components
│   └── admin/          # Admin-specific components
├── pages/              # Page components
│   ├── auth/           # Authentication pages
│   ├── profile/        # Profile pages
│   └── admin/          # Admin pages
├── contexts/           # React contexts
├── services/           # API services
├── types/              # TypeScript type definitions
└── utils/              # Utility functions
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint

## Pages & Routes

### Public Routes
- `/` - Home page
- `/login` - User login
- `/register` - User registration
- `/verify-email` - Email verification
- `/password-reset` - Password reset request
- `/reset-password` - Password reset confirmation

### Protected Routes
- `/profile` - User profile

### Admin Routes
- `/admin` - Admin dashboard
- `/admin/users` - User management
- `/admin/audit-logs` - Audit logs

## License

MIT License
