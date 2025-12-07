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
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import { apiClient } from '../../services/api';
import { SessionManagement } from '../../components/sessions/SessionManagement';
import type { ProfileUpdateRequest, User, ApiError } from '../../types';

export const ProfilePage = () => {
  const { user, updateUser } = useAuth();
  const [editModalOpened, setEditModalOpened] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

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
              {user.email_verified && (
                <Badge color="green" variant="light">Verified</Badge>
              )}
            </Group>
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

      <SessionManagement />
    </Container>
  );
};
