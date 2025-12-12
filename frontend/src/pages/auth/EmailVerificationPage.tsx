import { useEffect, useState, useRef } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Loader,
  Center,
  Stack,
} from '@mantine/core';
import { IconCheck, IconX } from '@tabler/icons-react';
import { authApi } from '../../services/api';
import type { ApiError } from '../../types';

export const EmailVerificationPage = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState('');
  const hasVerified = useRef(false);

  useEffect(() => {
    const token = searchParams.get('token');
    
    if (!token) {
      setStatus('error');
      setMessage('Verification token is missing');
      return;
    }

    // Prevent double execution in StrictMode
    if (hasVerified.current) {
      return;
    }
    hasVerified.current = true;

    const verifyEmail = async () => {
      try {
        const response = await authApi.verifyEmail(token);
        setStatus('success');
        setMessage(response.message || 'Email verified successfully!');
      } catch (error) {
        const apiError = error as ApiError;
        setStatus('error');
        setMessage(apiError.error || 'Failed to verify email');
      }
    };

    verifyEmail();
  }, [searchParams]);

  return (
    <Container size={420} my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Stack align="center" gap="md">
          {status === 'loading' && (
            <>
              <Loader size="xl" />
              <Title order={2}>Verifying Email...</Title>
              <Text c="dimmed" ta="center">
                Please wait while we verify your email address
              </Text>
            </>
          )}

          {status === 'success' && (
            <>
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
              <Title order={2}>Email Verified!</Title>
              <Text c="dimmed" ta="center">
                {message}
              </Text>
              <Button onClick={() => navigate('/login')} mt="md">
                Go to Login
              </Button>
            </>
          )}

          {status === 'error' && (
            <>
              <Center
                style={{
                  width: 80,
                  height: 80,
                  borderRadius: '50%',
                  backgroundColor: 'var(--mantine-color-red-1)',
                }}
              >
                <IconX size={48} color="var(--mantine-color-red-6)" />
              </Center>
              <Title order={2}>Verification Failed</Title>
              <Text c="dimmed" ta="center">
                {message}
              </Text>
              <Button onClick={() => navigate('/login')} variant="outline" mt="md">
                Go to Login
              </Button>
            </>
          )}
        </Stack>
      </Paper>
    </Container>
  );
};
