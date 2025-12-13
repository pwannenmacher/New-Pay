import { Container, Title, Text, Button, Stack, Paper } from '@mantine/core';
import { Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useAppConfig } from '../contexts/AppConfigContext';

export const HomePage = () => {
  const { isAuthenticated, user } = useAuth();
  const { enableRegistration } = useAppConfig();

  const hasAnyRole = user?.roles && user.roles.length > 0;
  const userRoles = user?.roles?.map(r => r.name).join(', ') || 'keine';

  return (
    <Container size="md" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Stack align="center" gap="lg">
          <Title order={1}>Willkommen bei New Pay</Title>
          
          {isAuthenticated ? (
            <>
              {!hasAnyRole ? (
                <>
                  <Text size="lg" ta="center" c="orange">
                    Hallo, {user?.first_name}!
                  </Text>
                  <Text size="md" ta="center">
                    Ihrem Benutzerkonto wurde noch keine Rolle zugewiesen.
                  </Text>
                  <Text size="sm" ta="center" c="dimmed">
                    Bitte kontaktieren Sie einen Administrator, um eine Rolle (user, reviewer oder admin) 
                    zugewiesen zu bekommen. Nur dann können Sie auf die Funktionen dieser Anwendung zugreifen.
                  </Text>
                  <Button component={Link} to="/profile" size="lg">
                    Zum Profil
                  </Button>
                </>
              ) : (
                <>
                  <Text size="lg" ta="center">
                    Hallo, {user?.first_name}! Willkommen auf Ihrer Plattform für Gehaltseinschätzungen und Peer-Reviews.
                  </Text>
                  <Button component={Link} to="/profile" size="lg">
                    View Profile
                  </Button>
                </>
              )}
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
