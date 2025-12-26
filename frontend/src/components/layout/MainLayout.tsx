import { AppShell, Burger, Group, Button, Menu, Avatar, Text, Alert, Divider } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Link, useNavigate } from 'react-router-dom';
import {
  IconUser,
  IconLogout,
  IconSettings,
  IconAlertCircle,
  IconMail,
  IconBook,
  IconClipboardList,
  IconUsers,
  IconShieldCheck,
  IconFileText,
  IconClock,
  IconCheckbox,
} from '@tabler/icons-react';
import { useAuth } from '../../contexts/AuthContext';
import { useAppConfig } from '../../contexts/AppConfigContext';
import { ThemeToggle } from './ThemeToggle';
import { useState } from 'react';
import { apiClient } from '../../services/api';
import { notifications } from '@mantine/notifications';

interface MainLayoutProps {
  children: React.ReactNode;
}

export const MainLayout = ({ children }: MainLayoutProps) => {
  const [opened, { toggle }] = useDisclosure();
  const { user, logout, isAuthenticated } = useAuth();
  const { enableRegistration } = useAppConfig();
  const navigate = useNavigate();
  const [isResendingVerification, setIsResendingVerification] = useState(false);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const handleResendVerification = async () => {
    setIsResendingVerification(true);
    try {
      await apiClient.post('/users/resend-verification', {});
      notifications.show({
        title: 'Success',
        message: 'Verification email sent successfully. Please check your inbox.',
        color: 'green',
      });
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: 'Failed to send verification email',
        color: 'red',
      });
    } finally {
      setIsResendingVerification(false);
    }
  };

  const isAdmin = user?.roles?.some((role) => role.name === 'admin');
  const isReviewer = user?.roles?.some((role) => role.name === 'reviewer');
  const hasUserRole = user?.roles?.some((role) => role.name === 'user');
  const hasAnyRole = user?.roles && user.roles.length > 0;

  return (
    <AppShell
      header={{ height: 60 }}
      navbar={{
        width: 300,
        breakpoint: 'sm',
        collapsed: { mobile: !opened },
      }}
      padding="md"
    >
      <AppShell.Header>
        <Group h="100%" px="md" justify="space-between">
          <Group>
            <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
            <Text
              size="xl"
              fw={700}
              component={Link}
              to="/"
              style={{ textDecoration: 'none', color: 'inherit' }}
            >
              New Pay
            </Text>
          </Group>

          {isAuthenticated ? (
            <Group>
              <ThemeToggle />
              <Menu shadow="md" width={200}>
                <Menu.Target>
                  <Button variant="subtle" leftSection={<Avatar size="sm" radius="xl" />}>
                    {user?.first_name}
                  </Button>
                </Menu.Target>

                <Menu.Dropdown>
                  <Menu.Label>{user?.email}</Menu.Label>
                  <Menu.Item leftSection={<IconUser size={14} />} component={Link} to="/profile">
                    Meine Daten
                  </Menu.Item>
                  <Menu.Item leftSection={<IconSettings size={14} />}>Einstellungen</Menu.Item>
                  <Menu.Divider />
                  <Menu.Item
                    color="red"
                    leftSection={<IconLogout size={14} />}
                    onClick={handleLogout}
                  >
                    Abmelden
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
          ) : (
            <Group>
              <ThemeToggle />
              <Button variant="subtle" component={Link} to="/login">
                Login
              </Button>
              {enableRegistration && (
                <Button component={Link} to="/register">
                  Sign Up
                </Button>
              )}
            </Group>
          )}
        </Group>
      </AppShell.Header>

      <AppShell.Navbar p="md">
        {isAuthenticated && (
          <>
            {/* Navigation for users with any role */}
            {hasAnyRole && (
              <>
                <Text size="sm" fw={500} mb="xs" c="dimmed">
                  Navigation
                </Text>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/profile"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconUser size={16} />}
                  mb="xs"
                >
                  Meine Daten
                </Button>
                {hasUserRole && (
                  <Button
                    variant="subtle"
                    component={Link}
                    to="/self-assessments"
                    fullWidth
                    justify="flex-start"
                    leftSection={<IconClipboardList size={16} />}
                    mb="xs"
                  >
                    Selbsteinschätzungen
                  </Button>
                )}
                {hasUserRole && (
                  <Button
                    variant="subtle"
                    component={Link}
                    to="/catalogs"
                    fullWidth
                    justify="flex-start"
                    leftSection={<IconBook size={16} />}
                    mb="xs"
                  >
                    Kriterienkataloge
                  </Button>
                )}

              </>
            )}

            {/* Show only profile link for users without roles */}
            {!hasAnyRole && (
              <>
                <Text size="sm" fw={500} mb="xs" c="dimmed">
                  Mein Profil
                </Text>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/profile"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconUser size={16} />}
                  mb="xs"
                >
                  Meine Daten
                </Button>
              </>
            )}

            {isReviewer && (
              <>
                <Divider my="md" />
                <Text size="sm" fw={500} mb="xs" c="dimmed">
                  Review
                </Text>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/review/open-assessments"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconClock size={16} />}
                  mb="xs"
                >
                  Offene Selbsteinschätzungen
                </Button>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/review/completed-assessments"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconFileText size={16} />}
                  mb="xs"
                  styles={{
                    root: { height: 'auto', padding: '8px 12px' },
                    inner: { whiteSpace: 'normal', justifyContent: 'flex-start' },
                    label: { whiteSpace: 'normal', wordBreak: 'break-word', textAlign: 'left' },
                  }}
                >
                  Abgeschlossene Selbsteinschätzungen
                </Button>
              </>
            )}

            {isAdmin && (
              <>
                <Divider my="md" />
                <Text size="sm" fw={500} mb="xs" c="dimmed">
                  Admin
                </Text>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin/catalogs"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconBook size={16} />}
                  mb="xs"
                >
                  Kriterienkataloge verwalten
                </Button>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin/self-assessments"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconClipboardList size={16} />}
                  mb="xs"
                >
                  Selbsteinschätzungen
                </Button>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin/users"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconUsers size={16} />}
                  mb="xs"
                >
                  Benutzerverwaltung
                </Button>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin/sessions"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconClock size={16} />}
                  mb="xs"
                >
                  Sitzungsverwaltung
                </Button>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin/audit-logs"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconFileText size={16} />}
                  mb="xs"
                >
                  Audit Logs
                </Button>
              </>
            )}
          </>
        )}
      </AppShell.Navbar>

      <AppShell.Main>
        {isAuthenticated && user && !user.email_verified && (
          <Alert
            variant="light"
            color="yellow"
            title="Email Verification Required"
            icon={<IconAlertCircle />}
            mb="md"
            withCloseButton={false}
          >
            <Group justify="space-between" align="center">
              <Text size="sm">
                Your email address has not been verified yet. Please check your inbox for the
                verification email.
              </Text>
              <Button
                size="xs"
                variant="light"
                color="yellow"
                leftSection={<IconMail size={14} />}
                onClick={handleResendVerification}
                loading={isResendingVerification}
              >
                Resend Email
              </Button>
            </Group>
          </Alert>
        )}
        {children}
      </AppShell.Main>
    </AppShell>
  );
};
