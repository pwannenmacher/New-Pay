import { Container, Title, Text, Button, Stack, Paper } from '@mantine/core';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useAppConfig } from '../contexts/AppConfigContext';

export const HomePage = () => {
  const { isAuthenticated, user } = useAuth();
  const { enableRegistration } = useAppConfig();

  return (
    <Container size="md" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Stack align="center" gap="lg">
          <Title order={1}>Welcome to New Pay</Title>
          
          {isAuthenticated ? (
            <>
              <Text size="lg" ta="center">
                Hello, {user?.first_name}! Welcome to your salary estimation and peer review platform.
              </Text>
              <Button component={Link} to="/profile" size="lg">
                View Profile
              </Button>
            </>
          ) : (
            <>
              <Text size="lg" ta="center" c="dimmed">
                A modern platform for salary estimates and peer reviews.
              </Text>
              <Text size="md" ta="center">
                {enableRegistration 
                  ? 'Sign up today to get started with salary insights and professional peer reviews.'
                  : 'Sign in to access salary insights and professional peer reviews.'}
              </Text>
              <Button.Group>
                {enableRegistration && (
                  <Button component={Link} to="/register" size="lg">
                    Get Started
                  </Button>
                )}
                <Button component={Link} to="/login" variant={enableRegistration ? "outline" : "filled"} size="lg">
                  Sign In
                </Button>
              </Button.Group>
            </>
          )}
        </Stack>
      </Paper>
    </Container>
  );
};
