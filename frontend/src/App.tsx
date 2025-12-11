import '@mantine/core/styles.css';
import '@mantine/notifications/styles.css';
import '@mantine/dates/styles.css';
import './index.css';
import { MantineProvider } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { DatesProvider } from '@mantine/dates';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { AppConfigProvider } from './contexts/AppConfigContext';
import { ThemeProvider } from './contexts/ThemeContext';
import { ProtectedRoute } from './components/auth/ProtectedRoute';
import { MainLayout } from './components/layout/MainLayout';
import { HomePage } from './pages/HomePage';
import { LoginPage } from './pages/auth/LoginPage';
import { RegisterPage } from './pages/auth/RegisterPage';
import { OAuthCallbackPage } from './pages/auth/OAuthCallbackPage';
import { EmailVerificationPage } from './pages/auth/EmailVerificationPage';
import { PasswordResetRequestPage } from './pages/auth/PasswordResetRequestPage';
import { PasswordResetConfirmPage } from './pages/auth/PasswordResetConfirmPage';
import { ProfilePage } from './pages/profile/ProfilePage';
import { UserManagementPage } from './pages/admin/UserManagementPage';
import { AuditLogsPage } from './pages/admin/AuditLogsPage';
import { SessionsPage } from './pages/admin/SessionsPage';
import { CatalogManagementPage } from './pages/admin/CatalogManagementPage';
import { CatalogEditorPage } from './pages/admin/CatalogEditorPage';
import { CatalogViewPage } from './pages/admin/CatalogViewPage';
import SelfAssessmentsPage from './pages/self-assessments/SelfAssessmentsPage';
import SelfAssessmentDetailPage from './pages/self-assessments/SelfAssessmentDetailPage';
import SelfAssessmentPage from './pages/SelfAssessmentPage';
import SelfAssessmentsAdminPage from './pages/admin/SelfAssessmentsAdminPage';

import 'dayjs/locale/de';

function App() {
  return (
    <MantineProvider defaultColorScheme="auto">
      <ThemeProvider>
        <DatesProvider settings={{ locale: 'de' }}>
          <Notifications position="top-right" />
      <BrowserRouter>
        <AppConfigProvider>
          <AuthProvider>
            <MainLayout>
              <Routes>
                <Route path="/" element={<HomePage />} />
                <Route path="/login" element={<LoginPage />} />
                <Route path="/register" element={<RegisterPage />} />
                <Route path="/oauth/callback" element={<OAuthCallbackPage />} />
                <Route path="/verify-email" element={<EmailVerificationPage />} />
                <Route path="/password-reset" element={<PasswordResetRequestPage />} />
                <Route path="/reset-password" element={<PasswordResetConfirmPage />} />
              
              <Route
                path="/profile"
                element={
                  <ProtectedRoute>
                    <ProfilePage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/users"
                element={
                  <ProtectedRoute requireAdmin>
                    <UserManagementPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/audit-logs"
                element={
                  <ProtectedRoute requireAdmin>
                    <AuditLogsPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/sessions"
                element={
                  <ProtectedRoute requireAdmin>
                    <SessionsPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/catalogs"
                element={
                  <ProtectedRoute requireAdmin>
                    <CatalogManagementPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/catalogs/:id"
                element={
                  <ProtectedRoute requireAdmin>
                    <CatalogViewPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/catalogs/:id/edit"
                element={
                  <ProtectedRoute requireAdmin>
                    <CatalogEditorPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/admin/self-assessments"
                element={
                  <ProtectedRoute requireAdmin>
                    <SelfAssessmentsAdminPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/self-assessments"
                element={
                  <ProtectedRoute>
                    <SelfAssessmentsPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/self-assessments/:id"
                element={
                  <ProtectedRoute>
                    <SelfAssessmentDetailPage />
                  </ProtectedRoute>
                }
              />
              
              <Route
                path="/self-assessments/:id/edit"
                element={
                  <ProtectedRoute>
                    <SelfAssessmentPage />
                  </ProtectedRoute>
                }
              />
            </Routes>
          </MainLayout>
          </AuthProvider>
        </AppConfigProvider>
      </BrowserRouter>
        </DatesProvider>
      </ThemeProvider>
    </MantineProvider>
  );
}

export default App;
