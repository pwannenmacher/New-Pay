import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Group,
  Button,
  Stack,
  Badge,
  Table,
  Modal,
  Alert,
  LoadingOverlay,
} from '@mantine/core';
import {
  IconPlus,
  IconFileCheck,
  IconClock,
  IconCheck,
  IconMessageCircle,
  IconArchive,
  IconX,
  IconAlertCircle,
} from '@tabler/icons-react';
import { selfAssessmentService } from '../../services/selfAssessment';
import type { SelfAssessment, CriteriaCatalog } from '../../types';
import { notifications } from '@mantine/notifications';
import { WeightedScoreBadge } from '../../components/WeightedScoreDisplay';

const statusConfig = {
  draft: { label: 'Entwurf', color: 'gray', icon: IconClock },
  submitted: { label: 'Eingereicht', color: 'blue', icon: IconFileCheck },
  in_review: { label: 'In Prüfung', color: 'yellow', icon: IconClock },
  reviewed: { label: 'Geprüft', color: 'orange', icon: IconCheck },
  discussion: { label: 'Besprechung', color: 'violet', icon: IconMessageCircle },
  archived: { label: 'Archiviert', color: 'green', icon: IconArchive },
  closed: { label: 'Geschlossen', color: 'red', icon: IconX },
};

export default function SelfAssessmentsPage() {
  const navigate = useNavigate();
  const [assessments, setAssessments] = useState<SelfAssessment[]>([]);
  const [activeCatalog, setActiveCatalog] = useState<CriteriaCatalog | null>(null);
  const [loading, setLoading] = useState(true);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [hasActiveAssessment, setHasActiveAssessment] = useState(false);

  useEffect(() => {
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [assessmentsData, catalogsData] = await Promise.all([
        selfAssessmentService.getMySelfAssessments(),
        selfAssessmentService.getActiveCatalogs(),
      ]);
      const assessmentsList = Array.isArray(assessmentsData) ? assessmentsData : [];
      const catalogsList = Array.isArray(catalogsData) ? catalogsData : [];

      setAssessments(assessmentsList);
      setActiveCatalog(catalogsList.length > 0 ? catalogsList[0] : null);

      // Check if user has any active assessment (not archived or closed)
      const activeStatuses = ['draft', 'submitted', 'in_review', 'reviewed', 'discussion'];
      const hasActive = assessmentsList.some((a) => activeStatuses.includes(a.status));
      setHasActiveAssessment(hasActive);
    } catch (error) {
      console.error('Error loading data:', error);
      setAssessments([]);
      setActiveCatalog(null);
      setHasActiveAssessment(false);
      notifications.show({
        title: 'Fehler',
        message: 'Daten konnten nicht geladen werden',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAssessment = async () => {
    if (!activeCatalog) return;

    try {
      setCreating(true);
      const newAssessment = await selfAssessmentService.createSelfAssessment(activeCatalog.id);
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde erstellt',
        color: 'green',
      });
      setCreateModalOpen(false);
      await loadData();
      navigate(`/self-assessments/${newAssessment.id}`);
    } catch (error: any) {
      console.error('Error creating assessment:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Selbsteinschätzung konnte nicht erstellt werden',
        color: 'red',
      });
    } finally {
      setCreating(false);
    }
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
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
      <Badge color={config.color} leftSection={<Icon size={14} />}>
        {config.label}
      </Badge>
    );
  };

  const canCreateAssessment = !hasActiveAssessment && activeCatalog !== null;
  const buttonTooltip = hasActiveAssessment
    ? 'Sie haben bereits eine aktive Selbsteinschätzung. Schließen oder archivieren Sie diese zuerst.'
    : !activeCatalog
      ? 'Zurzeit ist kein aktiver Kriterienkatalog verfügbar.'
      : '';

  if (loading) {
    return (
      <Container size="xl" py="xl">
        <LoadingOverlay visible={true} />
      </Container>
    );
  }

  return (
    <Container size="xl" py="xl">
      <Stack gap="lg">
        <Group justify="space-between">
          <div>
            <Title order={1}>Meine Selbsteinschätzungen</Title>
            <Text c="dimmed" size="sm">
              Verwalten Sie Ihre Selbsteinschätzungen zu aktiven Kriterienkatalogen
            </Text>
          </div>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => setCreateModalOpen(true)}
            disabled={!canCreateAssessment}
            title={buttonTooltip}
          >
            Neue Selbsteinschätzung
          </Button>
        </Group>

        {!canCreateAssessment && assessments.length === 0 && (
          <Alert icon={<IconAlertCircle size={16} />} title="Keine Kataloge verfügbar" color="blue">
            {buttonTooltip}
          </Alert>
        )}

        {assessments.length > 0 ? (
          <Paper shadow="sm" p="md" withBorder>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>Katalog</Table.Th>
                  <Table.Th>Status</Table.Th>
                  <Table.Th>Gesamtlevel</Table.Th>
                  <Table.Th>Erstellt am</Table.Th>
                  <Table.Th>Eingereicht am</Table.Th>
                  <Table.Th>Aktualisiert am</Table.Th>
                  <Table.Th>Aktionen</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {assessments.map((assessment) => (
                  <Table.Tr key={assessment.id}>
                    <Table.Td>
                      <Text size="sm">{assessment.catalog_name || 'Unbekannt'}</Text>
                    </Table.Td>
                    <Table.Td>{getStatusBadge(assessment.status)}</Table.Td>
                    <Table.Td>
                      <WeightedScoreBadge assessmentId={assessment.id} compact />
                    </Table.Td>
                    <Table.Td>{formatDate(assessment.created_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.submitted_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.updated_at)}</Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <Button
                          size="xs"
                          variant="light"
                          onClick={() => navigate(`/self-assessments/${assessment.id}`)}
                        >
                          Details
                        </Button>
                        {(assessment.status === 'discussion' || assessment.status === 'archived') && (
                          <Button
                            size="xs"
                            variant="light"
                            color="violet"
                            leftSection={<IconMessageCircle size={14} />}
                            onClick={() => navigate(`/discussion/${assessment.id}`)}
                          >
                            Besprechung
                          </Button>
                        )}
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </Paper>
        ) : (
          !loading && (
            <Paper shadow="sm" p="xl" withBorder>
              <Stack align="center" gap="md">
                <IconFileCheck size={48} stroke={1.5} />
                <Text c="dimmed">Sie haben noch keine Selbsteinschätzungen erstellt</Text>
                {activeCatalog && (
                  <Button
                    onClick={() => setCreateModalOpen(true)}
                    leftSection={<IconPlus size={16} />}
                  >
                    Erste Selbsteinschätzung erstellen
                  </Button>
                )}
              </Stack>
            </Paper>
          )
        )}
      </Stack>

      <Modal
        opened={createModalOpen}
        onClose={() => {
          setCreateModalOpen(false);
        }}
        title="Neue Selbsteinschätzung erstellen"
      >
        <Stack>
          {activeCatalog && (
            <div>
              <Text size="sm" fw={500} mb="xs">
                Aktiver Kriterienkatalog
              </Text>
              <Text size="sm">{activeCatalog.name}</Text>
              <Text size="xs" c="dimmed">
                Gültig von {formatDate(activeCatalog.valid_from)} bis{' '}
                {formatDate(activeCatalog.valid_until)}
              </Text>
            </div>
          )}

          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => setCreateModalOpen(false)} disabled={creating}>
              Abbrechen
            </Button>
            <Button onClick={handleCreateAssessment} disabled={!activeCatalog} loading={creating}>
              Erstellen
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Container>
  );
}
