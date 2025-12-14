import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Container,
  Title,
  Text,
  Paper,
  Group,
  Stack,
  Badge,
  Table,
  TextInput,
  Select,
  Button,
  LoadingOverlay,
} from '@mantine/core';
import {
  IconClock,
  IconFileCheck,
  IconCheck,
  IconMessageCircle,
  IconArchive,
  IconX,
  IconSearch,
  IconFilter,
} from '@tabler/icons-react';
import { DateInput } from '@mantine/dates';
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

export default function SelfAssessmentsAdminPage() {
  const navigate = useNavigate();
  const [assessments, setAssessments] = useState<SelfAssessment[]>([]);
  const [loading, setLoading] = useState(true);
  const [username, setUsername] = useState('');
  const [status, setStatus] = useState<string>('');
  const [fromDate, setFromDate] = useState<string | null>(null);
  const [toDate, setToDate] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const filters: any = {};
      if (status) filters.status = status;
      if (username) filters.username = username;
      if (fromDate) filters.from_date = new Date(fromDate).toISOString();
      if (toDate) filters.to_date = new Date(toDate).toISOString();

      const data = await selfAssessmentService.getAllSelfAssessmentsAdmin(filters);
      setAssessments(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Error loading data:', error);
      setAssessments([]);
      notifications.show({
        title: 'Fehler',
        message: 'Daten konnten nicht geladen werden',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleFilter = () => {
    loadData();
  };

  const handleReset = () => {
    setUsername('');
    setStatus('');
    setFromDate(null);
    setToDate(null);
    setTimeout(loadData, 100);
  };

  const handleClose = async (assessmentId: number) => {
    try {
      await selfAssessmentService.updateStatus(assessmentId, 'closed');
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde geschlossen',
        color: 'green',
      });
      await loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message:
          error.response?.data?.error || 'Selbsteinschätzung konnte nicht geschlossen werden',
        color: 'red',
      });
    }
  };

  const handleDelete = async (assessmentId: number) => {
    if (!confirm('Möchten Sie diese Selbsteinschätzung wirklich löschen?')) {
      return;
    }

    try {
      await selfAssessmentService.deleteSelfAssessment(assessmentId);
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde gelöscht',
        color: 'green',
      });
      await loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Selbsteinschätzung konnte nicht gelöscht werden',
        color: 'red',
      });
    }
  };

  const canDelete = (assessment: SelfAssessment) => {
    return assessment.status === 'closed' && !assessment.submitted_at;
  };

  const canReopen = (assessment: SelfAssessment) => {
    if (assessment.status !== 'closed' || !assessment.closed_at) {
      return false;
    }
    const closedAt = new Date(assessment.closed_at);
    const now = new Date();
    const hoursSinceClosed = (now.getTime() - closedAt.getTime()) / (1000 * 60 * 60);
    return hoursSinceClosed < 24;
  };

  const handleReopen = async (assessmentId: number, previousStatus: string | undefined) => {
    if (!previousStatus) {
      notifications.show({
        title: 'Fehler',
        message: 'Vorheriger Status nicht gefunden',
        color: 'red',
      });
      return;
    }

    try {
      await selfAssessmentService.updateStatus(assessmentId, previousStatus);
      notifications.show({
        title: 'Erfolg',
        message: 'Selbsteinschätzung wurde wiedereröffnet',
        color: 'green',
      });
      await loadData();
    } catch (error: any) {
      notifications.show({
        title: 'Fehler',
        message:
          error.response?.data?.error || 'Selbsteinschätzung konnte nicht wiedereröffnet werden',
        color: 'red',
      });
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

  return (
    <Container size="xl" py="xl">
      <LoadingOverlay visible={loading} />

      <Stack gap="lg">
        <div>
          <Title order={1}>Selbsteinschätzungen Verwaltung</Title>
          <Text c="dimmed" size="sm">
            Übersicht aller Selbsteinschätzungen mit Filtermöglichkeiten
          </Text>
        </div>

        <Paper shadow="sm" p="md" withBorder>
          <Stack gap="md">
            <Group align="flex-end" gap="md">
              <TextInput
                label="Username"
                placeholder="E-Mail, Vorname oder Nachname"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                leftSection={<IconSearch size={16} />}
                style={{ flex: 1 }}
              />
              <Select
                label="Status"
                placeholder="Alle Status"
                value={status}
                onChange={(value) => setStatus(value || '')}
                data={[
                  { value: '', label: 'Alle' },
                  { value: 'draft', label: 'Entwurf' },
                  { value: 'submitted', label: 'Eingereicht' },
                  { value: 'in_review', label: 'In Prüfung' },
                  { value: 'reviewed', label: 'Geprüft' },
                  { value: 'discussion', label: 'Besprechung' },
                  { value: 'archived', label: 'Archiviert' },
                  { value: 'closed', label: 'Geschlossen' },
                ]}
                clearable
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Von Datum"
                placeholder="Datum wählen"
                value={fromDate}
                onChange={(value) => setFromDate(value)}
                clearable
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Bis Datum"
                placeholder="Datum wählen"
                value={toDate}
                onChange={(value) => setToDate(value)}
                clearable
                style={{ minWidth: 200 }}
              />
            </Group>
            <Group>
              <Button leftSection={<IconFilter size={16} />} onClick={handleFilter}>
                Filtern
              </Button>
              <Button variant="subtle" onClick={handleReset}>
                Zurücksetzen
              </Button>
            </Group>
          </Stack>
        </Paper>

        {assessments.length > 0 ? (
          <Paper shadow="sm" p="md" withBorder>
            <Table striped highlightOnHover>
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>ID</Table.Th>
                  <Table.Th>Katalog</Table.Th>
                  <Table.Th>Benutzer</Table.Th>
                  <Table.Th>Status</Table.Th>
                  <Table.Th>Erstellt am</Table.Th>
                  <Table.Th>Eingereicht am</Table.Th>
                  <Table.Th>Aktualisiert am</Table.Th>
                  <Table.Th>Aktionen</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {assessments.map((assessment) => (
                  <Table.Tr key={assessment.id}>
                    <Table.Td>{assessment.id}</Table.Td>
                    <Table.Td>
                      <div>
                        <Text size="sm">{assessment.catalog_name || 'Unbekannt'}</Text>
                        <Text size="xs" c="dimmed">
                          ID: {assessment.catalog_id}
                        </Text>
                      </div>
                    </Table.Td>
                    <Table.Td>
                      <div>
                        <Text size="sm">{assessment.user_name || 'Unbekannt'}</Text>
                        <Text size="xs" c="dimmed">
                          ID: {assessment.user_id}
                        </Text>
                      </div>
                    </Table.Td>
                    <Table.Td>{getStatusBadge(assessment.status)}</Table.Td>
                    <Table.Td>{formatDate(assessment.created_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.submitted_at)}</Table.Td>
                    <Table.Td>{formatDate(assessment.updated_at)}</Table.Td>
                    <Table.Td>
                      <Group gap="xs">
                        <Button
                          size="xs"
                          variant="light"
                          onClick={() => navigate(`/admin/self-assessments/${assessment.id}`)}
                        >
                          Details
                        </Button>
                        {assessment.status !== 'closed' && (
                          <Button
                            size="xs"
                            variant="light"
                            color="red"
                            onClick={() => handleClose(assessment.id)}
                          >
                            Schließen
                          </Button>
                        )}
                        {canReopen(assessment) && (
                          <Button
                            size="xs"
                            variant="light"
                            color="orange"
                            onClick={() => handleReopen(assessment.id, assessment.previous_status)}
                          >
                            Wiedereröffnen
                          </Button>
                        )}
                        {canDelete(assessment) && (
                          <Button
                            size="xs"
                            variant="light"
                            color="red"
                            onClick={() => handleDelete(assessment.id)}
                          >
                            Löschen
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
                <Text c="dimmed">Keine Selbsteinschätzungen gefunden</Text>
              </Stack>
            </Paper>
          )
        )}
      </Stack>
    </Container>
  );
}
