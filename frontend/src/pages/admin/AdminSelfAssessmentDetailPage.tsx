import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Group,
  Button,
  Stack,
  Badge,
  Timeline,
  Alert,
  LoadingOverlay,
  Divider,
  Table,
} from '@mantine/core';
import {
  IconArrowLeft,
  IconClock,
  IconFileCheck,
  IconCheck,
  IconMessageCircle,
  IconArchive,
  IconX,
  IconAlertCircle,
} from '@tabler/icons-react';
import { selfAssessmentService } from '../../services/selfAssessment';
import adminService from '../../services/admin';
import type { SelfAssessment, CriteriaCatalog } from '../../types';
import { notifications } from '@mantine/notifications';

const statusConfig = {
  draft: { label: 'Entwurf', color: 'gray', icon: IconClock },
  submitted: { label: 'Eingereicht', color: 'blue', icon: IconFileCheck },
  in_review: { label: 'In Prüfung', color: 'yellow', icon: IconClock },
  reviewed: { label: 'Geprüft', color: 'orange', icon: IconCheck },
  discussion: { label: 'Besprechung', color: 'violet', icon: IconMessageCircle },
  archived: { label: 'Archiviert', color: 'green', icon: IconArchive },
  closed: { label: 'Geschlossen', color: 'red', icon: IconX },
};

export default function AdminSelfAssessmentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [assessment, setAssessment] = useState<SelfAssessment | null>(null);
  const [catalog, setCatalog] = useState<CriteriaCatalog | null>(null);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    if (id) {
      loadAssessment();
    }
  }, [id]);

  const loadAssessment = async () => {
    try {
      setLoading(true);
      const data = await selfAssessmentService.getSelfAssessment(parseInt(id!));
      setAssessment(data);

      // Load catalog information
      try {
        const catalogData = await adminService.getCatalog(data.catalog_id);
        setCatalog(catalogData);
      } catch (error) {
        console.error('Error loading catalog:', error);
      }
    } catch (error: any) {
      console.error('Error loading assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Selbsteinschätzung konnte nicht geladen werden',
        color: 'red',
      });
      navigate('/admin/self-assessments');
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = async (newStatus: string) => {
    if (!assessment) return;

    try {
      setUpdating(true);
      await selfAssessmentService.updateStatus(assessment.id, newStatus);
      notifications.show({
        title: 'Erfolg',
        message: `Status wurde auf "${statusConfig[newStatus as keyof typeof statusConfig]?.label || newStatus}" geändert`,
        color: 'green',
      });
      await loadAssessment();
    } catch (error: any) {
      console.error('Error updating status:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Status konnte nicht aktualisiert werden',
        color: 'red',
      });
    } finally {
      setUpdating(false);
    }
  };

  const handleDelete = async () => {
    if (!assessment) return;

    if (!confirm('Möchten Sie diese Selbsteinschätzung wirklich löschen?')) {
      return;
    }

    try {
      await selfAssessmentService.deleteSelfAssessment(assessment.id);
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde gelöscht',
        color: 'green',
      });
      navigate('/admin/self-assessments');
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Selbsteinschätzung konnte nicht gelöscht werden',
        color: 'red',
      });
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return null;
    return new Date(dateString).toLocaleString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusBadge = (status: string) => {
    const config = statusConfig[status as keyof typeof statusConfig] || {
      label: status,
      color: 'gray',
      icon: IconClock,
    };
    const Icon = config.icon;
    return (
      <Badge size="lg" color={config.color} leftSection={<Icon size={14} />}>
        {config.label}
      </Badge>
    );
  };

  const canDelete = () => {
    if (!assessment) return false;
    return assessment.status === 'closed' && !assessment.submitted_at;
  };

  const canReopen = () => {
    if (!assessment || assessment.status !== 'closed' || !assessment.closed_at) {
      return false;
    }
    const closedAt = new Date(assessment.closed_at);
    const now = new Date();
    const hoursSinceClosed = (now.getTime() - closedAt.getTime()) / (1000 * 60 * 60);
    return hoursSinceClosed < 24;
  };

  const getRemainingReopenTime = () => {
    if (!assessment?.closed_at) return null;
    const closedAt = new Date(assessment.closed_at);
    const deadline = new Date(closedAt.getTime() + 24 * 60 * 60 * 1000);
    const now = new Date();
    const remaining = deadline.getTime() - now.getTime();

    if (remaining <= 0) return null;

    const hours = Math.floor(remaining / (1000 * 60 * 60));
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  };

  const handleReopen = async () => {
    if (!assessment?.previous_status) {
      notifications.show({
        title: 'Fehler',
        message: 'Vorheriger Status nicht gefunden',
        color: 'red',
      });
      return;
    }

    try {
      setUpdating(true);
      await selfAssessmentService.updateStatus(assessment.id, assessment.previous_status);
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde wiedereröffnet',
        color: 'green',
      });
      await loadAssessment();
    } catch (error: any) {
      console.error('Error reopening assessment:', error);
      notifications.show({
        title: 'Fehler',
        message:
          error.response?.data?.error || 'Selbsteinschätzung konnte nicht wiedereröffnet werden',
        color: 'red',
      });
    } finally {
      setUpdating(false);
    }
  };

  if (loading) {
    return (
      <Container size="xl" py="xl">
        <LoadingOverlay visible />
      </Container>
    );
  }

  if (!assessment) {
    return null;
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Group>
            <Button
              variant="subtle"
              leftSection={<IconArrowLeft size={16} />}
              onClick={() => navigate('/admin/self-assessments')}
            >
              Zurück
            </Button>
            <div>
              <Title order={1}>
                {assessment.catalog_name || catalog?.name || 'Selbsteinschätzung'}
              </Title>
              <Text c="dimmed" size="sm">
                Benutzer: {assessment.user_name || 'Unbekannt'} (
                {assessment.user_email || 'keine E-Mail'})
              </Text>
            </div>
          </Group>
          {getStatusBadge(assessment.status)}
        </Group>

        <Paper shadow="sm" p="md" withBorder>
          <Stack gap="md">
            <div>
              <Text fw={500} size="sm" c="dimmed" mb={4}>
                Grundinformationen
              </Text>
              <Table withColumnBorders>
                <Table.Tbody>
                  <Table.Tr>
                    <Table.Td fw={500} w="200px">
                      Benutzer
                    </Table.Td>
                    <Table.Td>
                      {assessment.user_name || 'Unbekannt'} (ID: {assessment.user_id})
                    </Table.Td>
                  </Table.Tr>
                  <Table.Tr>
                    <Table.Td fw={500}>Katalog</Table.Td>
                    <Table.Td>
                      {assessment.catalog_name || catalog?.name || 'Unbekannt'} (ID:{' '}
                      {assessment.catalog_id})
                    </Table.Td>
                  </Table.Tr>
                  <Table.Tr>
                    <Table.Td fw={500}>Status</Table.Td>
                    <Table.Td>{getStatusBadge(assessment.status)}</Table.Td>
                  </Table.Tr>
                </Table.Tbody>
              </Table>
            </div>

            <Divider />

            <div>
              <Text fw={500} size="sm" c="dimmed" mb={4}>
                Zeitstempel
              </Text>
              <Group grow>
                <div>
                  <Text fw={500} size="sm" c="dimmed">
                    Erstellt am
                  </Text>
                  <Text>{formatDate(assessment.created_at)}</Text>
                </div>
                <div>
                  <Text fw={500} size="sm" c="dimmed">
                    Aktualisiert am
                  </Text>
                  <Text>{formatDate(assessment.updated_at)}</Text>
                </div>
              </Group>
            </div>

            <Divider />

            <div>
              <Text fw={500} size="sm" c="dimmed" mb="md">
                Administrative Aktionen
              </Text>

              <Stack gap="sm">
                {assessment.status === 'submitted' && (
                  <Group>
                    <Button
                      onClick={() => handleStatusChange('in_review')}
                      loading={updating}
                      color="yellow"
                    >
                      In Prüfung setzen
                    </Button>
                  </Group>
                )}

                {assessment.status === 'in_review' && (
                  <Group>
                    <Button
                      onClick={() => handleStatusChange('reviewed')}
                      loading={updating}
                      color="orange"
                    >
                      Als geprüft markieren
                    </Button>
                  </Group>
                )}

                {assessment.status === 'reviewed' && (
                  <Group>
                    <Button
                      onClick={() => handleStatusChange('discussion')}
                      loading={updating}
                      color="violet"
                    >
                      In Besprechung setzen
                    </Button>
                    <Button
                      onClick={() => handleStatusChange('archived')}
                      loading={updating}
                      color="green"
                    >
                      Archivieren
                    </Button>
                  </Group>
                )}

                {assessment.status === 'discussion' && (
                  <Group>
                    <Button
                      onClick={() => handleStatusChange('archived')}
                      loading={updating}
                      color="green"
                    >
                      Archivieren
                    </Button>
                  </Group>
                )}

                {assessment.status !== 'closed' && (
                  <Group>
                    <Button
                      onClick={() => handleStatusChange('closed')}
                      loading={updating}
                      variant="light"
                      color="red"
                    >
                      Schließen
                    </Button>
                  </Group>
                )}

                {canReopen() && (
                  <>
                    <Alert icon={<IconAlertCircle size={16} />} color="orange">
                      Diese Selbsteinschätzung wurde geschlossen. Sie kann innerhalb von 24 Stunden
                      nach dem Schließen wiedereröffnet werden.
                      {getRemainingReopenTime() && (
                        <Text size="sm" mt="xs" fw={500}>
                          Verbleibende Zeit: {getRemainingReopenTime()}
                        </Text>
                      )}
                    </Alert>
                    <Group>
                      <Button onClick={handleReopen} loading={updating} color="orange">
                        Wiedereröffnen
                      </Button>
                    </Group>
                  </>
                )}

                {canDelete() && (
                  <Group>
                    <Button onClick={handleDelete} variant="light" color="red">
                      Löschen
                    </Button>
                  </Group>
                )}
              </Stack>
            </div>
          </Stack>
        </Paper>

        <Paper shadow="sm" p="md" withBorder>
          <Title order={3} mb="md">
            Status-Historie
          </Title>
          <Timeline active={-1} bulletSize={24} lineWidth={2}>
            <Timeline.Item bullet={<IconClock size={12} />} title="Erstellt">
              <Text c="dimmed" size="sm">
                {formatDate(assessment.created_at)}
              </Text>
            </Timeline.Item>

            {assessment.submitted_at && (
              <Timeline.Item bullet={<IconFileCheck size={12} />} title="Eingereicht">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.submitted_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.in_review_at && (
              <Timeline.Item bullet={<IconClock size={12} />} title="In Prüfung">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.in_review_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.reviewed_at && (
              <Timeline.Item bullet={<IconCheck size={12} />} title="Geprüft">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.reviewed_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.discussion_started_at && (
              <Timeline.Item bullet={<IconMessageCircle size={12} />} title="Besprechung">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.discussion_started_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.archived_at && (
              <Timeline.Item bullet={<IconArchive size={12} />} title="Archiviert">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.archived_at)}
                </Text>
              </Timeline.Item>
            )}

            {assessment.closed_at && (
              <Timeline.Item bullet={<IconX size={12} />} title="Geschlossen">
                <Text c="dimmed" size="sm">
                  {formatDate(assessment.closed_at)}
                </Text>
                {assessment.previous_status && (
                  <Text c="dimmed" size="xs">
                    Vorheriger Status:{' '}
                    {statusConfig[assessment.previous_status as keyof typeof statusConfig]?.label ||
                      assessment.previous_status}
                  </Text>
                )}
              </Timeline.Item>
            )}
          </Timeline>
        </Paper>

        <Alert icon={<IconAlertCircle size={16} />} color="blue">
          <Text fw={500} mb="xs">
            Hinweis zu sensiblen Informationen
          </Text>
          <Text size="sm">
            Diese Ansicht zeigt keine Details zu den spezifischen Level-Einschätzungen des
            Benutzers, um die Vertraulichkeit der Selbsteinschätzung zu wahren. Als Administrator
            können Sie den Status verwalten und administrative Aktionen durchführen.
          </Text>
        </Alert>
      </Stack>
    </Container>
  );
}
