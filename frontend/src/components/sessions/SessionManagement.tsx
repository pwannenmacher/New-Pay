import { useState, useEffect } from 'react';
import { Paper, Title, Text, Button, Stack, Group, Card, Modal } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { IconDeviceLaptop, IconTrash, IconAlertTriangle } from '@tabler/icons-react';
import { sessionApi } from '../../services/api';
import type { Session, ApiError } from '../../types';

export const SessionManagement = () => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [deleteModalOpened, setDeleteModalOpened] = useState(false);
  const [deleteAllModalOpened, setDeleteAllModalOpened] = useState(false);
  const [sessionToDelete, setSessionToDelete] = useState<string | null>(null);

  const loadSessions = async () => {
    setIsLoading(true);
    try {
      const data = await sessionApi.getMySessions();
      setSessions(data);
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

  const handleDeleteSession = async () => {
    if (!sessionToDelete) return;

    try {
      await sessionApi.deleteMySession(sessionToDelete);
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

  const handleDeleteAllOtherSessions = async () => {
    try {
      await sessionApi.deleteAllMySessions();
      notifications.show({
        title: 'Success',
        message: 'All other sessions deleted successfully',
        color: 'green',
      });
      setDeleteAllModalOpened(false);
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
    return 'Unknown Browser';
  };

  return (
    <Paper withBorder shadow="md" p={30} radius="md" mt="lg">
      <Group justify="space-between" mb="xl">
        <Title order={3}>Active Sessions</Title>
        {sessions.length > 1 && (
          <Button
            color="red"
            variant="light"
            leftSection={<IconAlertTriangle size={16} />}
            onClick={() => setDeleteAllModalOpened(true)}
          >
            End All Other Sessions
          </Button>
        )}
      </Group>

      <Stack gap="md">
        {isLoading ? (
          <Text c="dimmed">Loading sessions...</Text>
        ) : sessions.length === 0 ? (
          <Text c="dimmed">No active sessions found</Text>
        ) : (
          sessions.map((session) => (
            <Card key={session.session_id} withBorder padding="md">
              <Group justify="space-between">
                <Group>
                  <IconDeviceLaptop size={24} />
                  <div>
                    <Text fw={500}>{getBrowserInfo(session.user_agent)}</Text>
                    <Text size="sm" c="dimmed">
                      {session.ip_address}
                    </Text>
                    <Text size="xs" c="dimmed">
                      Last active: {formatDate(session.last_activity_at)}
                    </Text>
                    <Text size="xs" c="dimmed">
                      Created: {formatDate(session.created_at)}
                    </Text>
                  </div>
                </Group>
                <Button
                  color="red"
                  variant="subtle"
                  leftSection={<IconTrash size={16} />}
                  onClick={() => {
                    setSessionToDelete(session.session_id);
                    setDeleteModalOpened(true);
                  }}
                >
                  End Session
                </Button>
              </Group>
            </Card>
          ))
        )}
      </Stack>

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
          Are you sure you want to end this session? You will be logged out from this device.
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
        onClose={() => setDeleteAllModalOpened(false)}
        title="End All Other Sessions"
        centered
      >
        <Text mb="md">
          Are you sure you want to end all other sessions? This will log you out from all other
          devices except this one.
        </Text>
        <Group justify="flex-end">
          <Button variant="subtle" onClick={() => setDeleteAllModalOpened(false)}>
            Cancel
          </Button>
          <Button color="red" onClick={handleDeleteAllOtherSessions}>
            End All Other Sessions
          </Button>
        </Group>
      </Modal>
    </Paper>
  );
};
