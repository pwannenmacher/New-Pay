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
import { useAppConfig } from '../../contexts/AppConfigContext';
import { useOAuthConfig } from '../../hooks/useOAuthConfig';
import type { LoginRequest, ApiError } from '../../types';

export const LoginPage = () => {
  const navigate = useNavigate();
  const { login } = useAuth();
  const { enableRegistration } = useAppConfig();
  const { config: oauthConfig } = useOAuthConfig();
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

  const handleOAuthLogin = (provider: string) => {
    const redirectUrl = `http://localhost:8080/api/v1/auth/oauth/login?provider=${encodeURIComponent(provider)}`;
    window.location.href = redirectUrl;
  };

  return (
    <Container size={420} my={40}>
      <Title ta="center" order={1}>
        Willkommen bei New Pay
      </Title>
      <Text c="dimmed" size="sm" ta="center" mt={5}>
        Melden Sie sich bei Ihrem Konto an
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
                Passwort vergessen?
              </Anchor>
            </Group>

            <Button type="submit" fullWidth mt="xl" loading={isLoading}>
              Anmelden
            </Button>
          </Stack>
        </form>

        {oauthConfig?.enabled && oauthConfig.providers.length > 0 && (
          <>
            <Divider label="Oder fortfahren mit" labelPosition="center" my="lg" />

            <Stack gap="xs">
              {oauthConfig.providers.map((provider) => (
                <Button
                  key={provider.name}
                  variant="default"
                  onClick={() => handleOAuthLogin(provider.name)}
                  fullWidth
                >
                  Sign in with {provider.name}
                </Button>
              ))}
            </Stack>
          </>
        )}

        {enableRegistration && (
          <Text ta="center" mt="md">
            Noch kein Konto?{' '}
            <Anchor component={Link} to="/register" weight={700}>
              Registrieren
            </Anchor>
          </Text>
        )}
      </Paper>
    </Container>
  );
};
