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
  IconSearch,
  IconFilter,
} from '@tabler/icons-react';
import { DateInput } from '@mantine/dates';
import { selfAssessmentService } from '../../services/selfAssessment';
import type { SelfAssessment, Role } from '../../types';
import { notifications } from '@mantine/notifications';
import { useAuth } from '../../contexts/AuthContext';

const statusConfig = {
  submitted: { label: 'Eingereicht', color: 'blue', icon: IconFileCheck },
  in_review: { label: 'In Prüfung', color: 'yellow', icon: IconClock },
  reviewed: { label: 'Geprüft', color: 'orange', icon: IconCheck },
  discussion: { label: 'Besprechung', color: 'violet', icon: IconMessageCircle },
};

export function ReviewOpenAssessmentsPage() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.roles?.some((role: Role) => role.name === 'admin');
  const [assessments, setAssessments] = useState<SelfAssessment[]>([]);
  const [allAssessments, setAllAssessments] = useState<SelfAssessment[]>([]);
  const [loading, setLoading] = useState(true);
  const [username, setUsername] = useState('');
  const [status, setStatus] = useState<string>('');
  const [catalogId, setCatalogId] = useState<string>('');
  const [fromDate, setFromDate] = useState<string | null>(null);
  const [toDate, setToDate] = useState<string | null>(null);
  const [fromSubmittedDate, setFromSubmittedDate] = useState<string | null>(null);
  const [toSubmittedDate, setToSubmittedDate] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const filters: any = {};
      if (status) filters.status = status;
      if (username) filters.username = username;
      if (catalogId) filters.catalog_id = parseInt(catalogId);
      if (fromDate) filters.from_date = fromDate;
      if (toDate) filters.to_date = toDate;
      if (fromSubmittedDate) filters.from_submitted_date = fromSubmittedDate;
      if (toSubmittedDate) filters.to_submitted_date = toSubmittedDate;

      const data = await selfAssessmentService.getOpenAssessmentsForReview(filters);
      const assessmentsList = Array.isArray(data) ? data : [];
      setAssessments(assessmentsList);
      
      // Store all assessments for catalog dropdown
      if (!allAssessments.length) {
        setAllAssessments(assessmentsList);
      }
    } catch (error: any) {
      console.error('Error loading assessments:', error);
      notifications.show({
        title: 'Fehler',
        message: error.response?.data?.error || 'Fehler beim Laden der Selbsteinschätzungen',
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
    setCatalogId('');
    setFromDate(null);
    setToDate(null);
    setFromSubmittedDate(null);
    setToSubmittedDate(null);
    // Reload with no filters
    setTimeout(() => loadData(), 100);
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleString('de-DE', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getStatusBadge = (status: string) => {
    const config = statusConfig[status as keyof typeof statusConfig];
    if (!config) return <Badge>{status}</Badge>;

    const Icon = config.icon;
    return (
      <Badge color={config.color} leftSection={<Icon size={14} />}>
        {config.label}
      </Badge>
    );
  };

  // Get unique catalogs from all assessments
  const catalogOptions = Array.from(
    new Map(
      allAssessments
        .filter((a) => a.catalog_id && a.catalog_name)
        .map((a) => [a.catalog_id, { value: String(a.catalog_id), label: a.catalog_name || `Katalog ${a.catalog_id}` }])
    ).values()
  );

  return (
    <Container size="xl" py="xl">
      <LoadingOverlay visible={loading} />

      <Stack gap="lg">
        <div>
          <Title order={1}>Offene Selbsteinschätzungen</Title>
          <Text c="dimmed" size="sm">
            Übersicht aller eingereichten Selbsteinschätzungen zur Prüfung
          </Text>
        </div>

        <Paper shadow="sm" p="md" withBorder>
          <Stack gap="md">
            <Group align="flex-end" gap="md">
              <TextInput
                label="Benutzer"
                placeholder="E-Mail, Vorname oder Nachname"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                leftSection={<IconSearch size={16} />}
                style={{ flex: 1 }}
              />
              <Select
                label="Katalog"
                placeholder="Alle Kataloge"
                value={catalogId}
                onChange={(value) => setCatalogId(value || '')}
                data={[
                  { value: '', label: 'Alle' },
                  ...catalogOptions,
                ]}
                clearable
                searchable
                style={{ minWidth: 200 }}
              />
              <Select
                label="Status"
                placeholder="Alle Status"
                value={status}
                onChange={(value) => setStatus(value || '')}
                data={[
                  { value: '', label: 'Alle' },
                  { value: 'submitted', label: 'Eingereicht' },
                  { value: 'in_review', label: 'In Prüfung' },
                  { value: 'reviewed', label: 'Geprüft' },
                  { value: 'discussion', label: 'Besprechung' },
                ]}
                clearable
                style={{ minWidth: 200 }}
              />
            </Group>
            <Group align="flex-end" gap="md">
              <DateInput
                label="Erstellt von"
                placeholder="Datum wählen"
                value={fromDate}
                onChange={setFromDate}
                clearable
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Erstellt bis"
                placeholder="Datum wählen"
                value={toDate}
                onChange={setToDate}
                clearable
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Eingereicht von"
                placeholder="Datum wählen"
                value={fromSubmittedDate}
                onChange={setFromSubmittedDate}
                clearable
                style={{ minWidth: 200 }}
              />
              <DateInput
                label="Eingereicht bis"
                placeholder="Datum wählen"
                value={toSubmittedDate}
                onChange={setToSubmittedDate}
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
                  {isAdmin && <Table.Th>ID</Table.Th>}
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
                    {isAdmin && <Table.Td>{assessment.id}</Table.Td>}
                    <Table.Td>
                      <div>
                        <Text size="sm">{assessment.catalog_name || 'Unbekannt'}</Text>
                        {isAdmin && (
                          <Text size="xs" c="dimmed">
                            ID: {assessment.catalog_id}
                          </Text>
                        )}
                      </div>
                    </Table.Td>
                    <Table.Td>
                      <div>
                        <Text size="sm">{assessment.user_name || 'Unbekannt'}</Text>
                        <Text size="xs" c="dimmed">
                          {assessment.user_email || 'keine E-Mail'}
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
                          onClick={() => navigate(`/review/assessment/${assessment.id}`)}
                          disabled={user?.id === assessment.user_id}
                        >
                          {user?.id === assessment.user_id ? 'Eigene Einschätzung' : 'Prüfen'}
                        </Button>
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
                <Text c="dimmed">Keine offenen Selbsteinschätzungen gefunden</Text>
              </Stack>
            </Paper>
          )
        )}
      </Stack>
    </Container>
  );
}
