import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Container, Loader, Text, Stack } from '@mantine/core';
import { useAuth } from '../../contexts/AuthContext';
import { apiClient } from '../../services/api';
import type { User } from '../../types';

export function OAuthCallbackPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { updateUser } = useAuth();

  useEffect(() => {
    const handleCallback = async () => {
      const accessToken = searchParams.get('access_token');
      const error = searchParams.get('error');

      if (error) {
        console.error('OAuth error:', error);
        navigate('/login?error=' + error);
        return;
      }

      if (!accessToken) {
        console.error('No access token received');
        navigate('/login?error=no_token');
        return;
      }

      // Store access token
      localStorage.setItem('access_token', accessToken);

      try {
        // Fetch user profile
        const userData = await apiClient.get<User>('/users/profile');
        updateUser(userData);
        
        // Redirect to home page
        navigate('/');
      } catch (error) {
        console.error('Failed to fetch user profile:', error);
        navigate('/login?error=profile_fetch_failed');
      }
    };

    handleCallback();
  }, [searchParams, navigate, updateUser]);

  return (
    <Container size={420} my={100}>
      <Stack align="center" gap="md">
        <Loader size="xl" />
        <Text size="lg">Completing sign in...</Text>
        <Text size="sm" c="dimmed">
          Please wait while we log you in
        </Text>
      </Stack>
    </Container>
  );
}
