import { useState, useEffect } from 'react';
import {
  Container,
  Paper,
  Title,
  Text,
  Button,
  Group,
  Table,
  Modal,
  Badge,
  TextInput,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconTrash, IconAlertTriangle, IconSearch } from '@tabler/icons-react';
import { adminApi } from '../../services/admin';
import type { Session, ApiError } from '../../types';

export const SessionsPage = () => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [filteredSessions, setFilteredSessions] = useState<Session[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [deleteModalOpened, setDeleteModalOpened] = useState(false);
  const [deleteAllModalOpened, setDeleteAllModalOpened] = useState(false);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);
  const [userToDeleteAll, setUserToDeleteAll] = useState<number | null>(null);

  const loadSessions = async () => {
    setIsLoading(true);
    try {
      const data = await adminApi.getAllSessions();
      setSessions(data);
      setFilteredSessions(data);
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to load sessions',
        color: 'red',
      });
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadSessions();
  }, []);

  useEffect(() => {
    if (searchQuery.trim() === '') {
      setFilteredSessions(sessions);
    } else {
      const query = searchQuery.toLowerCase();
      const filtered = sessions.filter(
        (session) =>
          session.user_email?.toLowerCase().includes(query) ||
          session.user_name?.toLowerCase().includes(query) ||
          session.ip_address.toLowerCase().includes(query)
      );
      setFilteredSessions(filtered);
    }
  }, [searchQuery, sessions]);

  const handleDeleteSession = async () => {
    if (!sessionToDelete) return;

    try {
      await adminApi.deleteUserSession(sessionToDelete);
      notifications.show({
        title: 'Success',
        message: 'Session deleted successfully',
        color: 'green',
      });
      setDeleteModalOpened(false);
      setSessionToDelete(null);
      loadSessions();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to delete session',
        color: 'red',
      });
    }
  };

  const handleDeleteAllUserSessions = async () => {
    if (!userToDeleteAll) return;

    try {
      await adminApi.deleteAllUserSessions(userToDeleteAll);
      notifications.show({
        title: 'Success',
        message: 'All user sessions deleted successfully',
        color: 'green',
      });
      setDeleteAllModalOpened(false);
      setUserToDeleteAll(null);
      loadSessions();
    } catch (error) {
      const apiError = error as ApiError;
      notifications.show({
        title: 'Error',
        message: apiError.error || 'Failed to delete sessions',
        color: 'red',
      });
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getBrowserInfo = (userAgent: string) => {
    if (userAgent.includes('Chrome')) return 'Chrome';
    if (userAgent.includes('Firefox')) return 'Firefox';
    if (userAgent.includes('Safari')) return 'Safari';
    if (userAgent.includes('Edge')) return 'Edge';
    return 'Unknown';
  };

  const getUserSessionCount = (userId: number) => {
    return sessions.filter((s) => s.user_id === userId).length;
  };

  return (
    <Container size="xl" my={40}>
      <Paper withBorder shadow="md" p={30} radius="md">
        <Group justify="space-between" mb="xl">
          <Title order={2}>Sitzungsverwaltung</Title>
          <Badge size="lg" variant="light">
            {searchQuery && filteredSessions.length !== sessions.length
              ? `${filteredSessions.length} von ${sessions.length} Sessions`
              : `${sessions.length} ${sessions.length === 1 ? 'Session' : 'Sessions'}`}
          </Badge>
        </Group>

        <TextInput
          placeholder="Search by user email, name, or IP address..."
          leftSection={<IconSearch size={16} />}
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.currentTarget.value)}
          mb="md"
        />

        {isLoading ? (
          <Text c="dimmed">Loading sessions...</Text>
        ) : filteredSessions.length === 0 ? (
          <Text c="dimmed">
            {searchQuery ? 'No sessions found matching your search' : 'No active sessions'}
          </Text>
        ) : (
          <Table striped highlightOnHover>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>User</Table.Th>
                <Table.Th>Email</Table.Th>
                <Table.Th>IP Address</Table.Th>
                <Table.Th>Browser</Table.Th>
                <Table.Th>Last Active</Table.Th>
                <Table.Th>Created</Table.Th>
                <Table.Th>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {filteredSessions.map((session) => (
                <Table.Tr key={session.session_id}>
                  <Table.Td>{session.user_name}</Table.Td>
                  <Table.Td>{session.user_email}</Table.Td>
                  <Table.Td>{session.ip_address}</Table.Td>
                  <Table.Td>{getBrowserInfo(session.user_agent)}</Table.Td>
                  <Table.Td>{formatDate(session.last_activity_at)}</Table.Td>
                  <Table.Td>{formatDate(session.created_at)}</Table.Td>
                  <Table.Td>
                    <Group gap="xs">
                      <Button
                        size="xs"
                        color="red"
                        variant="subtle"
                        leftSection={<IconTrash size={14} />}
                        onClick={() => {
                          setSessionToDelete(session.session_id);
                          setDeleteModalOpened(true);
                        }}
                      >
                        End
                      </Button>
                      {session.user_id && getUserSessionCount(session.user_id) > 1 && (
                        <Button
                          size="xs"
                          color="orange"
                          variant="subtle"
                          leftSection={<IconAlertTriangle size={14} />}
                          onClick={() => {
                            setUserToDeleteAll(session.user_id!);
                            setDeleteAllModalOpened(true);
                          }}
                        >
                          End All User
                        </Button>
                      )}
                    </Group>
                  </Table.Td>
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        )}
      </Paper>

      <Modal
        opened={deleteModalOpened}
        onClose={() => {
          setDeleteModalOpened(false);
          setSessionToDelete(null);
        }}
        title="End Session"
        centered
      >
        <Text mb="md">
          Are you sure you want to end this session? The user will be logged out from this device.
        </Text>
        <Group justify="flex-end">
          <Button
            variant="subtle"
            onClick={() => {
              setDeleteModalOpened(false);
              setSessionToDelete(null);
            }}
          >
            Cancel
          </Button>
          <Button color="red" onClick={handleDeleteSession}>
            End Session
          </Button>
        </Group>
      </Modal>

      <Modal
        opened={deleteAllModalOpened}
        onClose={() => {
          setDeleteAllModalOpened(false);
          setUserToDeleteAll(null);
        }}
        title="End All User Sessions"
        centered
      >
        <Text mb="md">
          Are you sure you want to end all sessions for this user? They will be logged out from all devices.
        </Text>
        <Group justify="flex-end">
          <Button
            variant="subtle"
            onClick={() => {
              setDeleteAllModalOpened(false);
              setUserToDeleteAll(null);
            }}
          >
            Cancel
          </Button>
          <Button color="red" onClick={handleDeleteAllUserSessions}>
            End All Sessions
          </Button>
        </Group>
      </Modal>
    </Container>
  );
};
