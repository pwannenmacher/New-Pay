import { useState } from 'react';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Stack,
  Group,
  Badge,
  TextInput,
  Modal,
  PasswordInput,
  Divider,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import { apiClient } from '../../services/api';
import { SessionManagement } from '../../components/sessions/SessionManagement';
import type { ProfileUpdateRequest, User, ApiError } from '../../types';

interface PasswordChangeRequest {
  current_password: string;
  new_password: string;
  confirm_password: string;
}

export const ProfilePage = () => {
  const { user, updateUser } = useAuth();
  const [editModalOpened, setEditModalOpened] = useState(false);
  const [passwordModalOpened, setPasswordModalOpened] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isPasswordLoading, setIsPasswordLoading] = useState(false);
  const [isResendingVerification, setIsResendingVerification] = useState(false);

  const form = useForm<ProfileUpdateRequest>({
    initialValues: {
      first_name: user?.first_name || '',
      last_name: user?.last_name || '',
    },
    validate: {
      first_name: (value) => (value.trim().length > 0 ? null : 'First name is required'),
      last_name: (value) => (value.trim().length > 0 ? null : 'Last name is required'),
    },
  });

  const passwordForm = useForm<PasswordChangeRequest>({
    initialValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
    },
    validate: {
      current_password: (value) => (value.length > 0 ? null : 'Current password is required'),
      new_password: (value) => {
        if (value.length < 8) {
          return 'Password must be at least 8 characters long';
        }
        return null;
      },
      confirm_password: (value, values) => {
        if (value !== values.new_password) {
          return 'Passwords do not match';
        }
        return null;
      },
    },
  });

  const handleUpdate = async (values: ProfileUpdateRequest) => {
    setIsLoading(true);
    
    try {
      const updatedUser = await apiClient.post<User>('/users/profile/update', values);
      updateUser(updatedUser);
      setEditModalOpened(false);
      notifications.show({
        title: 'Success',
        message: 'Profile updated successfully',
        color: 'green',
      });
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to update profile',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handlePasswordChange = async (values: PasswordChangeRequest) => {
    setIsPasswordLoading(true);
    
    try {
      await apiClient.post('/users/password/change', {
        current_password: values.current_password,
        new_password: values.new_password,
      });
      
      setPasswordModalOpened(false);
      passwordForm.reset();
      
      notifications.show({
        title: 'Success',
        message: 'Password changed successfully',
        color: 'green',
      });
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to change password',
        color: 'red',
      });
    } finally {
      setIsPasswordLoading(false);
    }
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
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to send verification email',
        color: 'red',
      });
    } finally {
      setIsResendingVerification(false);
    }
  };

  if (!user) {
    return null;
  }

  return (
    <Container size="md" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Group justify="space-between" mb="xl">
          <Title order={2}>My Profile</Title>
          <Button onClick={() => setEditModalOpened(true)}>
            Edit Profile
          </Button>
        </Group>

        <Stack gap="md">
          <div>
            <Text fw={500} size="sm" c="dimmed">Name</Text>
            <Text size="lg">{user.first_name} {user.last_name}</Text>
          </div>

          <div>
            <Text fw={500} size="sm" c="dimmed">Email</Text>
            <Group gap="xs">
              <Text size="lg">{user.email}</Text>
              {user.email_verified ? (
                <Badge color="green" variant="light">Verified</Badge>
              ) : (
                <Badge color="yellow" variant="light">Not Verified</Badge>
              )}
            </Group>
            {!user.email_verified && (
              <Button 
                size="xs" 
                variant="light" 
                mt="xs"
                onClick={handleResendVerification}
                loading={isResendingVerification}
              >
                Resend Verification Email
              </Button>
            )}
          </div>

          <div>
            <Text fw={500} size="sm" c="dimmed">Status</Text>
            <Badge color={user.is_active ? 'green' : 'red'} variant="light">
              {user.is_active ? 'Active' : 'Inactive'}
            </Badge>
          </div>

          {user.oauth_provider && (
            <div>
              <Text fw={500} size="sm" c="dimmed">OAuth Provider</Text>
              <Badge color="blue" variant="light">{user.oauth_provider}</Badge>
            </div>
          )}

          {user.oauth_connections && user.oauth_connections.length > 0 && (
            <div>
              <Text fw={500} size="sm" c="dimmed">Connected Accounts</Text>
              <Group gap="xs">
                {user.oauth_connections.map((conn) => (
                  <Badge key={conn.id} color="blue" variant="light">
                    {conn.provider}
                  </Badge>
                ))}
              </Group>
            </div>
          )}

          <div>
            <Text fw={500} size="sm" c="dimmed">Login Methods</Text>
            <Group gap="xs">
              {user.has_local_password && (
                <Badge color="gray" variant="light">Email + Password</Badge>
              )}
              {user.oauth_connections && user.oauth_connections.length > 0 ? (
                <Badge color="green" variant="light">
                  OAuth ({user.oauth_connections.length} provider{user.oauth_connections.length > 1 ? 's' : ''})
                </Badge>
              ) : (
                !user.has_local_password && (
                  <Badge color="yellow" variant="light">OAuth Only</Badge>
                )
              )}
            </Group>
          </div>

          {user.roles && user.roles.length > 0 && (
            <div>
              <Text fw={500} size="sm" c="dimmed">Roles</Text>
              <Group gap="xs">
                {user.roles.map((role) => (
                  <Badge key={role.id} color="violet" variant="light">
                    {role.name}
                  </Badge>
                ))}
              </Group>
            </div>
          )}

          <div>
            <Text fw={500} size="sm" c="dimmed">Member Since</Text>
            <Text size="lg">
              {new Date(user.created_at).toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'long',
                day: 'numeric',
              })}
            </Text>
          </div>

          {user.last_login_at && (
            <div>
              <Text fw={500} size="sm" c="dimmed">Last Login</Text>
              <Text size="lg">
                {new Date(user.last_login_at).toLocaleString('en-US', {
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </Text>
            </div>
          )}
        </Stack>
      </Paper>

      {user.has_local_password && (
        <Paper withBorder shadow="md" p={30} radius="md" mt="xl">
          <Group justify="space-between" mb="xl">
            <div>
              <Title order={2}>Security</Title>
              <Text size="sm" c="dimmed" mt={4}>Manage your password</Text>
            </div>
            <Button onClick={() => setPasswordModalOpened(true)}>
              Change Password
            </Button>
          </Group>

          <Text size="sm" c="dimmed">
            Last updated: {user.updated_at ? new Date(user.updated_at).toLocaleDateString() : 'Never'}
          </Text>
        </Paper>
      )}

      <Modal
        opened={editModalOpened}
        onClose={() => setEditModalOpened(false)}
        title="Edit Profile"
        centered
      >
        <form onSubmit={form.onSubmit(handleUpdate)}>
          <Stack>
            <TextInput
              label="First Name"
              placeholder="John"
              required
              {...form.getInputProps('first_name')}
            />
            <TextInput
              label="Last Name"
              placeholder="Doe"
              required
              {...form.getInputProps('last_name')}
            />
            <Button type="submit" loading={isLoading} fullWidth>
              Save Changes
            </Button>
          </Stack>
        </form>
      </Modal>

      <Modal
        opened={passwordModalOpened}
        onClose={() => {
          setPasswordModalOpened(false);
          passwordForm.reset();
        }}
        title="Change Password"
        centered
      >
        <form onSubmit={passwordForm.onSubmit(handlePasswordChange)}>
          <Stack>
            <PasswordInput
              label="Current Password"
              placeholder="Enter your current password"
              required
              {...passwordForm.getInputProps('current_password')}
            />
            <Divider />
            <PasswordInput
              label="New Password"
              placeholder="Enter new password (min. 8 characters)"
              required
              {...passwordForm.getInputProps('new_password')}
            />
            <PasswordInput
              label="Confirm New Password"
              placeholder="Re-enter new password"
              required
              {...passwordForm.getInputProps('confirm_password')}
            />
            <Button type="submit" loading={isPasswordLoading} fullWidth>
              Change Password
            </Button>
          </Stack>
        </form>
      </Modal>

      <SessionManagement />
    </Container>
  );
};
