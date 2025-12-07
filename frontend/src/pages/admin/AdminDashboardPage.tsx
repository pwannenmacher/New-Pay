import { Container, Paper, Title, SimpleGrid, Card, Text, Group } from '@mantine/core';
import { IconUsers, IconFileText, IconDevices } from '@tabler/icons-react';
import { Link } from 'react-router-dom';

export const AdminDashboardPage = () => {
  const adminCards = [
    {
      title: 'User Management',
      description: 'Manage users and assign roles',
      icon: IconUsers,
      link: '/admin/users',
      color: 'blue',
    },
    {
      title: 'Session Management',
      description: 'View and manage active user sessions',
      icon: IconDevices,
      link: '/admin/sessions',
      color: 'green',
    },
    {
      title: 'Audit Logs',
      description: 'View security and activity logs',
      icon: IconFileText,
      link: '/admin/audit-logs',
      color: 'orange',
    },
  ];

  return (
    <Container size="xl" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Title order={2} mb="xl">Admin Dashboard</Title>
        
        <SimpleGrid cols={{ base: 1, sm: 2, lg: 3 }} spacing="lg">
          {adminCards.map((card) => (
            <Card
              key={card.title}
              component={Link}
              to={card.link}
              shadow="sm"
              padding="lg"
              radius="md"
              withBorder
              style={{ cursor: 'pointer', textDecoration: 'none' }}
            >
              <Group>
                <card.icon size={40} color={`var(--mantine-color-${card.color}-6)`} />
                <div style={{ flex: 1 }}>
                  <Text fw={500} size="lg">{card.title}</Text>
                  <Text size="sm" c="dimmed">{card.description}</Text>
                </div>
              </Group>
            </Card>
          ))}
        </SimpleGrid>
      </Paper>
    </Container>
  );
};
