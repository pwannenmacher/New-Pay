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
  Select,
  Modal,
  Alert,
  LoadingOverlay,
  Divider,
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
  IconSend,
} from '@tabler/icons-react';
import { selfAssessmentService } from '../../services/selfAssessment';
import type { SelfAssessment } from '../../types';
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

const allowedTransitions: Record<string, string[]> = {
  draft: ['submitted', 'closed'],
  submitted: [],
  in_review: [],
  reviewed: [],
  discussion: [],
  archived: [],
  closed: ['draft', 'submitted', 'in_review', 'reviewed', 'discussion'],
};

export default function SelfAssessmentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [assessment, setAssessment] = useState<SelfAssessment | null>(null);
  const [loading, setLoading] = useState(true);
  const [statusModalOpen, setStatusModalOpen] = useState(false);
  const [selectedStatus, setSelectedStatus] = useState<string | null>(null);
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
    } catch (error: any) {
      console.error('Error loading assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Selbsteinschätzung konnte nicht geladen werden',
        color: 'red',
      });
      navigate('/self-assessments');
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = async () => {
    if (!selectedStatus || !assessment) return;

    try {
      setUpdating(true);
      await selfAssessmentService.updateStatus(assessment.id, selectedStatus);
      notifications.show({
        title: 'Erfolg',
        message: 'Status wurde aktualisiert',
        color: 'green',
      });
      setStatusModalOpen(false);
      setSelectedStatus(null);
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

  const getAvailableTransitions = () => {
    if (!assessment) return [];
    const transitions = allowedTransitions[assessment.status] || [];
    
    // Check for closed status 24h restriction
    if (assessment.status === 'closed' && assessment.closed_at) {
      const closedTime = new Date(assessment.closed_at).getTime();
      const now = new Date().getTime();
      const hoursSinceClosed = (now - closedTime) / (1000 * 60 * 60);
      
      if (hoursSinceClosed > 24) {
        return [];
      }
    }
    
    return transitions;
  };

  const canSubmit = assessment?.status === 'draft';

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

  const availableTransitions = getAvailableTransitions();

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <Group>
            <Button
              variant="subtle"
              leftSection={<IconArrowLeft size={16} />}
              onClick={() => navigate('/self-assessments')}
            >
              Zurück
            </Button>
            <div>
              <Title order={1}>Selbsteinschätzung #{assessment.id}</Title>
              <Text c="dimmed" size="sm">
                Katalog ID: {assessment.catalog_id}
              </Text>
            </div>
          </Group>
          {getStatusBadge(assessment.status)}
        </Group>

        <Paper shadow="sm" p="md" withBorder>
          <Stack gap="md">
            <div>
              <Text fw={500} size="sm" c="dimmed" mb={4}>
                Aktueller Status
              </Text>
              {getStatusBadge(assessment.status)}
            </div>

            <Divider />

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

            {canSubmit && (
              <>
                <Divider />
                <Alert icon={<IconAlertCircle size={16} />} color="blue">
                  Diese Selbsteinschätzung befindet sich noch im Entwurf. Sie können sie bearbeiten
                  und dann zur Prüfung einreichen.
                </Alert>
                <Button
                  leftSection={<IconSend size={16} />}
                  onClick={() => {
                    setSelectedStatus('submitted');
                    setStatusModalOpen(true);
                  }}
                >
                  Zur Prüfung einreichen
                </Button>
              </>
            )}

            {availableTransitions.length > 0 && !canSubmit && (
              <>
                <Divider />
                <Button variant="light" onClick={() => setStatusModalOpen(true)}>
                  Status ändern
                </Button>
              </>
            )}
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
                    Vorheriger Status: {statusConfig[assessment.previous_status as keyof typeof statusConfig]?.label || assessment.previous_status}
                  </Text>
                )}
              </Timeline.Item>
            )}
          </Timeline>
        </Paper>
      </Stack>

      <Modal
        opened={statusModalOpen}
        onClose={() => {
          setStatusModalOpen(false);
          setSelectedStatus(null);
        }}
        title="Status ändern"
      >
        <Stack>
          <Select
            label="Neuer Status"
            placeholder="Wählen Sie einen Status"
            data={availableTransitions.map((status) => ({
              value: status,
              label: statusConfig[status as keyof typeof statusConfig]?.label || status,
            }))}
            value={selectedStatus}
            onChange={setSelectedStatus}
          />

          {assessment.status === 'closed' && assessment.closed_at && (
            <Alert icon={<IconAlertCircle size={16} />} color="orange">
              Diese Selbsteinschätzung wurde geschlossen. Sie können sie innerhalb von 24 Stunden
              in den vorherigen Status zurückversetzen.
            </Alert>
          )}

          <Group justify="flex-end" mt="md">
            <Button
              variant="subtle"
              onClick={() => {
                setStatusModalOpen(false);
                setSelectedStatus(null);
              }}
              disabled={updating}
            >
              Abbrechen
            </Button>
            <Button onClick={handleStatusChange} disabled={!selectedStatus} loading={updating}>
              Ändern
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}
