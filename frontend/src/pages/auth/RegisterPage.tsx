import { useState, useEffect } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Text,
  Container,
  Anchor,
  Stack,
  Group,
  Divider,
  Alert,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import { useAppConfig } from '../../contexts/AppConfigContext';
import { useOAuthConfig } from '../../hooks/useOAuthConfig';
import type { RegisterRequest, ApiError } from '../../types';

export const RegisterPage = () => {
  const navigate = useNavigate();
  const { register } = useAuth();
  const { enableRegistration, loading: configLoading } = useAppConfig();
  const { config: oauthConfig } = useOAuthConfig();
  const [isLoading, setIsLoading] = useState(false);

  // Redirect if registration is disabled
  useEffect(() => {
    if (!configLoading && !enableRegistration) {
      navigate('/login');
    }
  }, [enableRegistration, configLoading, navigate]);

  const form = useForm<RegisterRequest>({
    initialValues: {
      email: '',
      password: '',
      first_name: '',
      last_name: '',
    },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value) ? null : 'Invalid email'),
      password: (value) => (value.length >= 8 ? null : 'Password must be at least 8 characters'),
      first_name: (value) => (value.trim().length > 0 ? null : 'First name is required'),
      last_name: (value) => (value.trim().length > 0 ? null : 'Last name is required'),
    },
  });

  const handleSubmit = async (values: RegisterRequest) => {
    setIsLoading(true);

    try {
      await register(values);
      notifications.show({
        title: 'Success',
        message: 'Account created successfully! Please check your email to verify your account.',
        color: 'green',
      });
      navigate('/');
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Registration failed',
        message: apiError.error || 'Could not create account',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleOAuthLogin = (provider: string) => {
    const redirectUrl = `http://localhost:8080/api/v1/auth/oauth/login?provider=${encodeURIComponent(provider)}`;
    window.location.href = redirectUrl;
  };

  // Show loading or nothing while checking config
  if (configLoading) {
    return null;
  }

  // If registration is disabled, show message (should redirect anyway)
  if (!enableRegistration) {
    return (
      <Container size={420} my={40}>
        <Alert color="yellow" title="Registration Disabled">
          Registration is currently disabled. Please contact an administrator.
        </Alert>
      </Container>
    );
  }

  return (
    <Container size={420} my={40}>
      <Title ta="center" order={1}>
        Create Account
      </Title>
      <Text c="dimmed" size="sm" ta="center" mt={5}>
        Join New Pay to get started
      </Text>

      <Paper withBorder shadow="md" p={30} mt={30} radius="md">
        <form onSubmit={form.onSubmit(handleSubmit)}>
          <Stack>
            <Group grow>
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
            </Group>

            <TextInput
              label="Email"
              placeholder="you@example.com"
              required
              {...form.getInputProps('email')}
            />

            <PasswordInput
              label="Password"
              placeholder="At least 8 characters"
              required
              {...form.getInputProps('password')}
            />

            <Button type="submit" fullWidth mt="xl" loading={isLoading}>
              Create account
            </Button>
          </Stack>
        </form>

        {oauthConfig?.enabled && oauthConfig.providers.length > 0 && (
          <>
            <Divider label="Or continue with" labelPosition="center" my="lg" />

            <Stack gap="xs">
              {oauthConfig.providers.map((provider) => (
                <Button
                  key={provider.name}
                  variant="default"
                  onClick={() => handleOAuthLogin(provider.name)}
                  fullWidth
                >
                  Sign up with {provider.name}
                </Button>
              ))}
            </Stack>
          </>
        )}

        <Text ta="center" mt="md">
          Already have an account?{' '}
          <Anchor component={Link} to="/login" fw={700}>
            Sign in
          </Anchor>
        </Text>
      </Paper>
    </Container>
  );
};
