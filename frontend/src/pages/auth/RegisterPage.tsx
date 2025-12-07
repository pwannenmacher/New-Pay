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
  Anchor,
  Stack,
  Group,
  Divider,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';
import type { RegisterRequest, ApiError } from '../../types';

export const RegisterPage = () => {
  const navigate = useNavigate();
  const { register } = useAuth();
  const [isLoading, setIsLoading] = useState(false);

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

  const handleOAuthLogin = (provider: 'google' | 'facebook') => {
    const redirectUrl = `http://localhost:8080/api/v1/auth/${provider}/login`;
    window.location.href = redirectUrl;
  };

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
          Already have an account?{' '}
          <Anchor component={Link} to="/login" fw={700}>
            Sign in
          </Anchor>
        </Text>
      </Paper>
    </Container>
  );
};
