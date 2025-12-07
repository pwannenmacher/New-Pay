import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Text,
  Container,
  Group,
  Anchor,
  Stack,
  Divider,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import type { LoginRequest, ApiError } from '../../types';

export const LoginPage = () => {
  const navigate = useNavigate();
  const { login } = useAuth();
  const [isLoading, setIsLoading] = useState(false);

  const form = useForm<LoginRequest>({
    initialValues: {
      email: '',
      password: '',
    },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value) ? null : 'Invalid email'),
      password: (value) => (value.length >= 8 ? null : 'Password must be at least 8 characters'),
    },
  });

  const handleSubmit = async (values: LoginRequest) => {
    setIsLoading(true);
    
    try {
      await login(values);
      notifications.show({
        title: 'Success',
        message: 'Logged in successfully',
        color: 'green',
      });
      navigate('/');
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Login failed',
        message: apiError.error || 'Invalid credentials',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleOAuthLogin = (provider: 'google' | 'facebook') => {
    // Redirect to OAuth provider
    const redirectUrl = `http://localhost:8080/api/v1/auth/${provider}/login`;
    window.location.href = redirectUrl;
  };

  return (
    <Container size={420} my={40}>
      <Title ta="center" order={1}>
        Welcome to New Pay
      </Title>
      <Text c="dimmed" size="sm" ta="center" mt={5}>
        Sign in to your account
      </Text>

      <Paper withBorder shadow="md" p={30} mt={30} radius="md">
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <Stack>
            <TextInput
              label="Email"
              placeholder="you@example.com"
              required
              {...form.getInputProps('email')}
            />

            <PasswordInput
              label="Password"
              placeholder="Your password"
              required
              {...form.getInputProps('password')}
            />

            <Group justify="space-between" mt="md">
              <Anchor component={Link} to="/password-reset" size="sm">
                Forgot password?
              </Anchor>
            </Group>

            <Button type="submit" fullWidth mt="xl" loading={isLoading}>
              Sign in
            </Button>
          </Stack>
        </form>

        <Divider label="Or continue with" labelPosition="center" my="lg" />

        <Group grow mb="md" mt="md">
          <Button
            variant="default"
            onClick={() => handleOAuthLogin('google')}
          >
            Google
          </Button>
          <Button
            variant="default"
            onClick={() => handleOAuthLogin('facebook')}
          >
            Facebook
          </Button>
        </Group>

        <Text ta="center" mt="md">
          Don&apos;t have an account?{' '}
          <Anchor component={Link} to="/register" fw={700}>
            Register
          </Anchor>
        </Text>
      </Paper>
    </Container>
  );
};
