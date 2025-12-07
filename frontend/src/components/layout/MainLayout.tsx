import { AppShell, Burger, Group, Button, Menu, Avatar, Text, Alert } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Link, useNavigate } from 'react-router-dom';
import { IconUser, IconLogout, IconSettings, IconShieldLock, IconAlertCircle, IconMail, IconBook } from '@tabler/icons-react';
import { useAuth } from '../../contexts/AuthContext';
import { useAppConfig } from '../../contexts/AppConfigContext';
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
            <Text size="xl" fw={700} component={Link} to="/" style={{ textDecoration: 'none', color: 'inherit' }}>
              New Pay
            </Text>
          </Group>

          {isAuthenticated ? (
            <Menu shadow="md" width={200}>
              <Menu.Target>
                <Button variant="subtle" leftSection={<Avatar size="sm" radius="xl" />}>
                  {user?.first_name}
                </Button>
              </Menu.Target>

              <Menu.Dropdown>
                <Menu.Label>{user?.email}</Menu.Label>
                <Menu.Item
                  leftSection={<IconUser size={14} />}
                  component={Link}
                  to="/profile"
                >
                  Profile
                </Menu.Item>
                {isAdmin && (
                  <Menu.Item
                    leftSection={<IconShieldLock size={14} />}
                    component={Link}
                    to="/admin"
                  >
                    Admin Dashboard
                  </Menu.Item>
                )}
                <Menu.Item leftSection={<IconSettings size={14} />}>
                  Settings
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item
                  color="red"
                  leftSection={<IconLogout size={14} />}
                  onClick={handleLogout}
                >
                  Logout
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          ) : (
            <Group>
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
        <Text size="sm" fw={500} mb="md">
          Navigation
        </Text>
        {isAuthenticated && (
          <>
            <Button
              variant="subtle"
              component={Link}
              to="/profile"
              fullWidth
              justify="flex-start"
              mb="xs"
            >
              My Profile
            </Button>
            {isAdmin && (
              <>
                <Text size="sm" fw={500} mt="md" mb="md">
                  Admin
                </Text>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin"
                  fullWidth
                  justify="flex-start"
                  mb="xs"
                >
                  Admin Dashboard
                </Button>
                <Button
                  variant="subtle"
                  component={Link}
                  to="/admin/catalogs"
                  fullWidth
                  justify="flex-start"
                  leftSection={<IconBook size={16} />}
                  mb="xs"
                >
                  Kriterienkataloge
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
                Your email address has not been verified yet. Please check your inbox for the verification email.
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
