import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Paper,
  TextInput,
  Button,
  Title,
  Text,
  Container,
  Anchor,
  Stack,
  Center,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { IconCheck } from '@tabler/icons-react';
import { authApi } from '../../services/api';
import type { PasswordResetRequest, ApiError } from '../../types';

export const PasswordResetRequestPage = () => {
  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);

  const form = useForm<PasswordResetRequest>({
    initialValues: {
      email: '',
    },
    validate: {
      email: (value) => (/^\S+@\S+$/.test(value) ? null : 'Invalid email'),
    },
  });

  const handleSubmit = async (values: PasswordResetRequest) => {
    setIsLoading(true);
    
    try {
      await authApi.requestPasswordReset(values);
      setIsSuccess(true);
      notifications.show({
        title: 'Success',
        message: 'Password reset instructions sent to your email',
        color: 'green',
      });
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to send reset email',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

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
            <Title order={2}>Prüfen Sie Ihre E-Mails</Title>
            <Text c="dimmed" ta="center">
              Wir haben Ihnen Anweisungen zum Zurücksetzen des Passworts an Ihre E-Mail-Adresse gesendet.
              Bitte überprüfen Sie Ihren Posteingang und folgen Sie dem Link.
            </Text>
            <Anchor component={Link} to="/login" mt="md">
              Zurück zur Anmeldung
            </Anchor>
          </Stack>
        </Paper>
      </Container>
    );
  }

  return (
    <Container size={420} my={40}>
      <Title ta="center" order={1}>
        Passwort zurücksetzen
      </Title>
      <Text c="dimmed" size="sm" ta="center" mt={5}>
        Geben Sie Ihre E-Mail-Adresse ein, um Anweisungen zu erhalten
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

            <Button type="submit" fullWidth mt="xl" loading={isLoading}>
              Send reset instructions
            </Button>
          </Stack>
        </form>

        <Text ta="center" mt="md">
          <Anchor component={Link} to="/login" size="sm">
            Back to login
          </Anchor>
        </Text>
      </Paper>
    </Container>
  );
};
