import { useState } from 'react';
import { useSearchParams, useNavigate, Link } from 'react-router-dom';
import {
  Paper,
  PasswordInput,
  Button,
  Title,
  Text,
  Container,
  Stack,
  Center,
  Anchor,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { IconCheck } from '@tabler/icons-react';
import { authApi } from '../../services/api';
import type { ApiError } from '../../types';

interface PasswordResetForm {
  password: string;
  confirmPassword: string;
}

export const PasswordResetConfirmPage = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const token = searchParams.get('token');

  const form = useForm<PasswordResetForm>({
    initialValues: {
      password: '',
      confirmPassword: '',
    },
    validate: {
      password: (value) => (value.length >= 8 ? null : 'Password must be at least 8 characters'),
      confirmPassword: (value, values) =>
        value === values.password ? null : 'Passwords do not match',
    },
  });

  const handleSubmit = async (values: PasswordResetForm) => {
    if (!token) {
      notifications.show({
        title: 'Error',
        message: 'Reset token is missing',
        color: 'red',
      });
      return;
    }

    setIsLoading(true);

    try {
      await authApi.confirmPasswordReset({
        token,
        new_password: values.password,
      });
      setIsSuccess(true);
      notifications.show({
        title: 'Success',
        message: 'Password reset successfully',
        color: 'green',
      });
      setTimeout(() => navigate('/login'), 2000);
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to reset password',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  if (!token) {
    return (
      <Container size={420} my={40}>
        <Paper withBorder shadow="md" p={30} radius="md">
          <Stack align="center" gap="md">
            <Title order={2}>Ungültiger Link</Title>
            <Text c="dimmed" ta="center">
              This password reset link is invalid or has expired.
            </Text>
            <Anchor component={Link} to="/password-reset">
              Request a new reset link
            </Anchor>
          </Stack>
        </Paper>
      </Container>
    );
  }

  if (isSuccess) {
    return (
      <Container size={420} my={40}>
        <Paper withBorder shadow="md" p={30} radius="md">
          <Stack align="center" gap="md">
            <Center
              style={{
                width: 80,
                height: 80,
                borderRadius: '50%',
                backgroundColor: 'var(--mantine-color-green-1)',
              }}
            >
              <IconCheck size={48} color="var(--mantine-color-green-6)" />
            </Center>
            <Title order={2}>Passwort erfolgreich zurückgesetzt</Title>
            <Text c="dimmed" ta="center">
              Your password has been successfully reset. You will be redirected to the login page.
            </Text>
          </Stack>
        </Paper>
      </Container>
    );
  }

  return (
    <Container size={420} my={40}>
      <Title ta="center" order={1}>
        Neues Passwort festlegen
      </Title>
      <Text c="dimmed" size="sm" ta="center" mt={5}>
        Geben Sie Ihr neues Passwort ein
      </Text>

      <Paper withBorder shadow="md" p={30} mt={30} radius="md">
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <Stack>
            <PasswordInput
              label="New Password"
              placeholder="At least 8 characters"
              required
              {...form.getInputProps('password')}
            />

            <PasswordInput
              label="Confirm Password"
              placeholder="Re-enter your password"
              required
              {...form.getInputProps('confirmPassword')}
            />

            <Button type="submit" fullWidth mt="xl" loading={isLoading}>
              Reset password
            </Button>
          </Stack>
        </form>
      </Paper>
    </Container>
  );
};
